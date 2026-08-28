// Command migrate-v2 migrates legacy reference mappings and records affected by
// the old zero-code/ObjectID bugs. It is dry-run by default and refuses to apply
// a plan if the online snapshot changed or any conflict was reported.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Boyuan-IT-Club/Meowpick-Backend/infra/model"
	typemapping "github.com/Boyuan-IT-Club/Meowpick-Backend/types/mapping"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const planVersion = 2

type plan struct {
	Version              int             `json:"version"`
	GeneratedAt          time.Time       `json:"generatedAt"`
	Database             string          `json:"database"`
	SnapshotSHA256       string          `json:"snapshotSha256"`
	PlanSHA256           string          `json:"planSha256"`
	Deployment           string          `json:"deployment"`
	Conflicts            []string        `json:"conflicts"`
	Warnings             []string        `json:"warnings"`
	Mappings             []planMapping   `json:"mappings"`
	CourseRepairs        []courseRepair  `json:"courseRepairs"`
	CommentIDRepairs     []commentRepair `json:"commentIdRepairs"`
	TeacherTimeRepairs   []teacherRepair `json:"teacherTimeRepairs"`
	LegacyMappingIDCount int             `json:"legacyMappingIdCount"`
}

type planMapping struct {
	Type      model.MappingType `json:"type"`
	Name      string            `json:"name"`
	Code      int32             `json:"code"`
	Canonical bool              `json:"canonical"`
}

type courseRepair struct {
	ID         string  `json:"id"`
	Department *int32  `json:"department,omitempty"`
	Category   *int32  `json:"category,omitempty"`
	Campuses   []int32 `json:"campuses,omitempty"`
}

type commentRepair struct {
	ObjectID string `json:"objectId"`
	StringID string `json:"stringId"`
}

type teacherRepair struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type legacyMapping struct {
	ID   any               `bson:"_id"`
	Type model.MappingType `bson:"type"`
	Name string            `bson:"name"`
	Code int32             `bson:"code"`
}

type brokenCourse struct {
	ID         string  `bson:"_id"`
	ProposalID string  `bson:"proposalId"`
	Department int32   `bson:"department"`
	Category   int32   `bson:"category"`
	Campuses   []int32 `bson:"campuses"`
}

type proposalSource struct {
	Course struct {
		Department string   `bson:"department"`
		Category   string   `bson:"category"`
		Campuses   []string `bson:"campuses"`
	} `bson:"course"`
}

func main() {
	var uri, database, reportPath, applyPlanPath string
	flag.StringVar(&uri, "uri", "", "MongoDB URI (or MEOWPICK_MONGO_URI)")
	flag.StringVar(&database, "db", "meowpick", "MongoDB database")
	flag.StringVar(&reportPath, "report", "migration-v2-plan.json", "dry-run report path")
	flag.StringVar(&applyPlanPath, "apply-plan", "", "apply this previously generated dry-run plan")
	flag.Parse()
	if uri == "" {
		uri = os.Getenv("MEOWPICK_MONGO_URI")
	}
	if uri == "" {
		fatal(errors.New("--uri or MEOWPICK_MONGO_URI is required"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		fatal(err)
	}
	defer client.Disconnect(context.Background())
	if err = client.Ping(ctx, nil); err != nil {
		fatal(err)
	}

	current, err := buildPlan(ctx, client, database)
	if err != nil {
		fatal(err)
	}
	if applyPlanPath == "" {
		current.PlanSHA256, err = planHash(current)
		if err != nil {
			fatal(err)
		}
		if err = writePlan(reportPath, current); err != nil {
			fatal(err)
		}
		fmt.Printf("dry-run complete: %s\nconflicts=%d mappings=%d courseRepairs=%d commentIdRepairs=%d teacherTimeRepairs=%d\n",
			reportPath, len(current.Conflicts), len(current.Mappings), len(current.CourseRepairs), len(current.CommentIDRepairs), len(current.TeacherTimeRepairs))
		if len(current.Conflicts) > 0 {
			os.Exit(2)
		}
		return
	}

	approved, err := readPlan(applyPlanPath)
	if err != nil {
		fatal(err)
	}
	if approved.Version != planVersion || approved.Database != database {
		fatal(errors.New("plan version or database does not match"))
	}
	expectedPlanHash, err := planHash(approved)
	if err != nil || approved.PlanSHA256 == "" || approved.PlanSHA256 != expectedPlanHash {
		fatal(errors.New("plan checksum is invalid; use an unmodified dry-run report"))
	}
	if len(approved.Conflicts) > 0 {
		fatal(errors.New("plan contains conflicts; resolve them and run dry-run again"))
	}
	if len(current.Conflicts) > 0 {
		fatal(fmt.Errorf("current database has conflicts; new dry-run required: %s", strings.Join(current.Conflicts, "; ")))
	}
	if approved.SnapshotSHA256 != current.SnapshotSHA256 {
		fatal(errors.New("database changed after dry-run; refusing apply, generate a new plan"))
	}
	if err = apply(ctx, client, approved); err != nil {
		fatal(err)
	}
	fmt.Println("migration applied successfully; restart the backend to atomically warm Redis")
}

func buildPlan(ctx context.Context, client *mongo.Client, database string) (*plan, error) {
	db := client.Database(database)
	result := &plan{Version: planVersion, GeneratedAt: time.Now().UTC(), Database: database}
	result.Deployment, result.Conflicts = deployment(ctx, client)

	// Startup creates a case-insensitive partial unique nickname index. Detect
	// legacy collisions before deployment so startup cannot fail unexpectedly.
	usernameCursor, err := db.Collection("user").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"username": bson.M{"$type": "string", "$gt": ""}}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$toLower": bson.M{"$trim": bson.M{"input": "$username"}}},
			"ids": bson.M{"$push": "$_id"}, "count": bson.M{"$sum": 1},
		}}},
		{{Key: "$match", Value: bson.M{"count": bson.M{"$gt": 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var usernameConflicts []struct {
		Username string   `bson:"_id"`
		IDs      []string `bson:"ids"`
	}
	if err = usernameCursor.All(ctx, &usernameConflicts); err != nil {
		return nil, err
	}
	for _, conflict := range usernameConflicts {
		result.Conflicts = append(result.Conflicts,
			fmt.Sprintf("case-insensitive username conflict username=%q userIds=%v", conflict.Username, conflict.IDs))
	}
	negativeCursor, err := db.Collection("user").Find(ctx, bson.M{"contributionPoints": bson.M{"$lt": 0}},
		options.Find().SetProjection(bson.M{"_id": 1, "contributionPoints": 1}))
	if err != nil {
		return nil, err
	}
	var negativeContributions []struct {
		ID           string `bson:"_id"`
		Contribution int64  `bson:"contributionPoints"`
	}
	if err = negativeCursor.All(ctx, &negativeContributions); err != nil {
		return nil, err
	}
	for _, user := range negativeContributions {
		result.Conflicts = append(result.Conflicts,
			fmt.Sprintf("negative contribution userId=%s contributionPoints=%d", user.ID, user.Contribution))
	}

	var existing []legacyMapping
	cursor, err := db.Collection("mapping").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(ctx, &existing); err != nil {
		return nil, err
	}
	for _, item := range existing {
		if _, ok := item.ID.(primitive.ObjectID); !ok {
			result.LegacyMappingIDCount++
		}
	}

	byName, byCode := map[string]planMapping{}, map[string]planMapping{}
	staticCodesByName := map[string]map[int32]struct{}{}
	addSeeds := func(mappingType model.MappingType, seeds map[int32]string) {
		codes := make([]int, 0, len(seeds))
		for code := range seeds {
			codes = append(codes, int(code))
		}
		sort.Ints(codes)
		for _, code := range codes {
			name := strings.TrimSpace(seeds[int32(code)])
			item := planMapping{Type: mappingType, Name: name, Code: int32(code)}
			nameKey, codeKey := mappingNameKey(mappingType, name), mappingCodeKey(mappingType, int32(code))
			if staticCodesByName[nameKey] == nil {
				staticCodesByName[nameKey] = map[int32]struct{}{}
			}
			staticCodesByName[nameKey][int32(code)] = struct{}{}
			if current, ok := byName[nameKey]; !ok || item.Code < current.Code {
				item.Canonical = true
				if ok {
					current.Canonical = false
					byCode[mappingCodeKey(current.Type, current.Code)] = current
				}
				byName[nameKey] = item
			}
			byCode[codeKey] = item
		}
	}
	addSeeds(model.MappingTypeDepartment, typemapping.DepartmentsMap)
	addSeeds(model.MappingTypeCategory, typemapping.CategoriesMap)
	addSeeds(model.MappingTypeCampus, typemapping.CampusesMap)

	onlineByName := map[string]int32{}
	for _, legacy := range existing {
		item := planMapping{Type: legacy.Type, Name: strings.TrimSpace(legacy.Name), Code: legacy.Code, Canonical: true}
		if item.Type < model.MappingTypeDepartment || item.Type > model.MappingTypeCampus || item.Code <= 0 || item.Name == "" {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("invalid online mapping: type=%d code=%d name=%q", item.Type, item.Code, item.Name))
			continue
		}
		nameKey, codeKey := mappingNameKey(item.Type, item.Name), mappingCodeKey(item.Type, item.Code)
		if static, ok := byCode[codeKey]; ok && static.Name != item.Name {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("code conflict type=%d code=%d online=%q static=%q", item.Type, item.Code, item.Name, static.Name))
			continue
		}
		if allowedCodes, ok := staticCodesByName[nameKey]; ok {
			if _, allowed := allowedCodes[item.Code]; !allowed {
				result.Conflicts = append(result.Conflicts, fmt.Sprintf("name conflict type=%d name=%q onlineCode=%d staticCodes=%v", item.Type, item.Name, item.Code, sortedCodes(allowedCodes)))
			}
			continue
		}
		if oldCode, ok := onlineByName[nameKey]; ok && oldCode != item.Code {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("online name conflict type=%d name=%q codes=%d,%d", item.Type, item.Name, oldCode, item.Code))
			continue
		}
		if old, ok := byCode[codeKey]; ok && old.Name != item.Name {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("online code conflict type=%d code=%d names=%q,%q", item.Type, item.Code, old.Name, item.Name))
			continue
		}
		onlineByName[nameKey] = item.Code
		byName[nameKey], byCode[codeKey] = item, item
	}

	maxCode := map[model.MappingType]int32{}
	for _, item := range byCode {
		if item.Code > maxCode[item.Type] {
			maxCode[item.Type] = item.Code
		}
	}
	resolveDynamic := func(mappingType model.MappingType, name string) int32 {
		name = strings.TrimSpace(name)
		if item, ok := byName[mappingNameKey(mappingType, name)]; ok {
			return item.Code
		}
		maxCode[mappingType]++
		if name == "" {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("empty repair mapping name for type=%d", mappingType))
			return 0
		}
		item := planMapping{Type: mappingType, Name: name, Code: maxCode[mappingType], Canonical: true}
		byName[mappingNameKey(mappingType, name)] = item
		byCode[mappingCodeKey(mappingType, item.Code)] = item
		return item.Code
	}

	broken := []brokenCourse{}
	cursor, err = db.Collection("course").Find(ctx, bson.M{"$or": bson.A{
		bson.M{"department": bson.M{"$lte": 0}},
		bson.M{"category": bson.M{"$lte": 0}},
		bson.M{"campuses": bson.M{"$elemMatch": bson.M{"$lte": 0}}},
	}})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(ctx, &broken); err != nil {
		return nil, err
	}
	for _, course := range broken {
		if course.ProposalID == "" {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("course %s has zero mapping code without proposalId", course.ID))
			continue
		}
		var source proposalSource
		if err = db.Collection("proposal").FindOne(ctx, bson.M{"_id": course.ProposalID}).Decode(&source); err != nil {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("course %s cannot load proposal %s", course.ID, course.ProposalID))
			continue
		}
		repair := courseRepair{ID: course.ID}
		if course.Department <= 0 {
			code := resolveDynamic(model.MappingTypeDepartment, source.Course.Department)
			repair.Department = &code
		}
		if course.Category <= 0 {
			code := resolveDynamic(model.MappingTypeCategory, source.Course.Category)
			repair.Category = &code
		}
		if containsNonPositive(course.Campuses) {
			repair.Campuses = make([]int32, 0, len(source.Course.Campuses))
			for _, campus := range source.Course.Campuses {
				item, ok := byName[mappingNameKey(model.MappingTypeCampus, strings.TrimSpace(campus))]
				if !ok {
					result.Conflicts = append(result.Conflicts, fmt.Sprintf("course %s references unknown campus %q", course.ID, campus))
					continue
				}
				repair.Campuses = append(repair.Campuses, item.Code)
			}
		}
		result.CourseRepairs = append(result.CourseRepairs, repair)
	}

	commentCursor, err := db.Collection("comment").Find(ctx, bson.M{"_id": bson.M{"$type": "objectId"}}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	var objectIDComments []struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err = commentCursor.All(ctx, &objectIDComments); err != nil {
		return nil, err
	}
	stringCursor, err := db.Collection("comment").Find(ctx, bson.M{"_id": bson.M{"$type": "string"}}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	var stringComments []struct {
		ID string `bson:"_id"`
	}
	if err = stringCursor.All(ctx, &stringComments); err != nil {
		return nil, err
	}
	stringCommentIDs := make(map[string]struct{}, len(stringComments))
	for _, comment := range stringComments {
		stringCommentIDs[comment.ID] = struct{}{}
	}
	for _, comment := range objectIDComments {
		hexID := comment.ID.Hex()
		if _, exists := stringCommentIDs[hexID]; exists {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("comment id collision for %s", hexID))
			continue
		}
		result.CommentIDRepairs = append(result.CommentIDRepairs, commentRepair{ObjectID: hexID, StringID: hexID})
	}

	teacherCursor, err := db.Collection("teacher").Find(ctx, bson.M{"createdAt": bson.M{"$lt": time.Date(1972, 1, 1, 0, 0, 0, 0, time.UTC)}})
	if err != nil {
		return nil, err
	}
	var teachers []struct {
		ID string `bson:"_id"`
	}
	if err = teacherCursor.All(ctx, &teachers); err != nil {
		return nil, err
	}
	for _, teacher := range teachers {
		objectID, parseErr := primitive.ObjectIDFromHex(teacher.ID)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("teacher %s has 1970 timestamp but ID cannot supply recovery time", teacher.ID))
			continue
		}
		timestamp := objectID.Timestamp().UTC()
		result.TeacherTimeRepairs = append(result.TeacherTimeRepairs, teacherRepair{ID: teacher.ID, CreatedAt: timestamp, UpdatedAt: timestamp})
	}

	duplicatePipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"proposalId": bson.M{"$type": "string", "$gt": ""}}}},
		{{Key: "$group", Value: bson.M{"_id": "$proposalId", "count": bson.M{"$sum": 1}}}},
		{{Key: "$match", Value: bson.M{"count": bson.M{"$gt": 1}}}},
	}
	duplicateCursor, err := db.Collection("course").Aggregate(ctx, duplicatePipeline)
	if err != nil {
		return nil, err
	}
	var duplicates []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err = duplicateCursor.All(ctx, &duplicates); err != nil {
		return nil, err
	}
	for _, duplicate := range duplicates {
		result.Conflicts = append(result.Conflicts, fmt.Sprintf("proposal %s is linked to %d courses", duplicate.ID, duplicate.Count))
	}

	for _, item := range byCode {
		result.Mappings = append(result.Mappings, item)
	}
	sort.Slice(result.Mappings, func(i, j int) bool {
		if result.Mappings[i].Type == result.Mappings[j].Type {
			return result.Mappings[i].Code < result.Mappings[j].Code
		}
		return result.Mappings[i].Type < result.Mappings[j].Type
	})
	sort.Strings(result.Conflicts)
	sort.Strings(result.Warnings)
	result.SnapshotSHA256, err = snapshotHash(existing, broken, result.CourseRepairs, result.CommentIDRepairs, result.TeacherTimeRepairs)
	return result, err
}

func deployment(ctx context.Context, client *mongo.Client) (string, []string) {
	var hello bson.M
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil {
		return "unknown", []string{"cannot run MongoDB hello preflight: " + err.Error()}
	}
	if msg, _ := hello["msg"].(string); msg == "isdbgrid" {
		return "sharded", nil
	}
	if setName, _ := hello["setName"].(string); setName != "" {
		return "replicaSet:" + setName, nil
	}
	return "standalone", []string{"MongoDB deployment is standalone; proposal approval and migration require replica-set or sharded transactions"}
}

func snapshotHash(mappings []legacyMapping, courses []brokenCourse, repairs []courseRepair, comments []commentRepair, teachers []teacherRepair) (string, error) {
	parts := make([]string, 0, len(mappings)+len(courses)+len(repairs)+len(comments)+len(teachers))
	for _, item := range mappings {
		parts = append(parts, fmt.Sprintf("m|%v|%d|%d|%s", item.ID, item.Type, item.Code, item.Name))
	}
	for _, item := range courses {
		parts = append(parts, fmt.Sprintf("c|%s|%s|%d|%d|%v", item.ID, item.ProposalID, item.Department, item.Category, item.Campuses))
	}
	// Repairs include values resolved from the source proposal. Hashing them makes
	// apply reject a plan when that proposal changes after dry-run, even if the
	// broken course document itself stayed unchanged.
	for _, item := range repairs {
		parts = append(parts, fmt.Sprintf("r|%s|%s|%s|%v", item.ID, optionalInt32(item.Department), optionalInt32(item.Category), item.Campuses))
	}
	for _, item := range comments {
		parts = append(parts, "o|"+item.ObjectID)
	}
	for _, item := range teachers {
		parts = append(parts, "t|"+item.ID)
	}
	sort.Strings(parts)
	hash := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(hash[:]), nil
}

func optionalInt32(value *int32) string {
	if value == nil {
		return "-"
	}
	return strconv.FormatInt(int64(*value), 10)
}

func planHash(value *plan) (string, error) {
	copy := *value
	copy.PlanSHA256 = ""
	data, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func apply(ctx context.Context, client *mongo.Client, approved *plan) error {
	db := client.Database(approved.Database)
	// An early development build used a fully unique name index, which cannot
	// represent historical aliases. The v2 canonical partial index replaces it.
	if _, err := db.Collection("mapping").Indexes().DropOne(ctx, "mapping_type_name_unique"); err != nil {
		var commandError mongo.CommandError
		if !errors.As(err, &commandError) || (commandError.Code != 26 && commandError.Code != 27) {
			return fmt.Errorf("drop obsolete mapping name index: %w", err)
		}
	}
	session, err := client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx mongo.SessionContext) (any, error) {
		mappingCollection := db.Collection("mapping")
		if _, err := mappingCollection.DeleteMany(tx, bson.M{}); err != nil {
			return nil, err
		}
		if len(approved.Mappings) > 0 {
			documents := make([]any, 0, len(approved.Mappings))
			for _, item := range approved.Mappings {
				documents = append(documents, bson.M{"_id": primitive.NewObjectID(), "type": item.Type, "name": item.Name, "code": item.Code, "canonical": item.Canonical})
			}
			if _, err := mappingCollection.InsertMany(tx, documents); err != nil {
				return nil, err
			}
		}
		counterCollection := db.Collection("mapping_counter")
		if _, err := counterCollection.DeleteMany(tx, bson.M{}); err != nil {
			return nil, err
		}
		maxCode := map[model.MappingType]int32{}
		for _, item := range approved.Mappings {
			if item.Code > maxCode[item.Type] {
				maxCode[item.Type] = item.Code
			}
		}
		for mappingType, code := range maxCode {
			if _, err := counterCollection.InsertOne(tx, bson.M{"_id": mappingType, "seq": code}); err != nil {
				return nil, err
			}
		}

		for _, repair := range approved.CourseRepairs {
			set := bson.M{}
			if repair.Department != nil {
				set["department"] = *repair.Department
			}
			if repair.Category != nil {
				set["category"] = *repair.Category
			}
			if repair.Campuses != nil {
				set["campuses"] = repair.Campuses
			}
			if len(set) > 0 {
				res, updateErr := db.Collection("course").UpdateOne(tx, bson.M{"_id": repair.ID}, bson.M{"$set": set})
				if updateErr != nil || res.MatchedCount != 1 {
					return nil, fmt.Errorf("repair course %s matched=%d: %w", repair.ID, res.MatchedCount, updateErr)
				}
			}
		}
		for _, repair := range approved.CommentIDRepairs {
			objectID, parseErr := primitive.ObjectIDFromHex(repair.ObjectID)
			if parseErr != nil {
				return nil, parseErr
			}
			var document bson.M
			if err := db.Collection("comment").FindOne(tx, bson.M{"_id": objectID}).Decode(&document); err != nil {
				return nil, err
			}
			document["_id"] = repair.StringID
			if _, err := db.Collection("comment").InsertOne(tx, document); err != nil {
				return nil, err
			}
			if _, err := db.Collection("comment").DeleteOne(tx, bson.M{"_id": objectID}); err != nil {
				return nil, err
			}
		}
		for _, repair := range approved.TeacherTimeRepairs {
			if _, err := db.Collection("teacher").UpdateOne(tx, bson.M{"_id": repair.ID}, bson.M{"$set": bson.M{"createdAt": repair.CreatedAt, "updatedAt": repair.UpdatedAt}}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	if err != nil {
		return err
	}

	if _, err = db.Collection("mapping").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "type", Value: 1}, {Key: "name", Value: 1}}, Options: options.Index().SetName("mapping_type_name_canonical_unique").SetUnique(true).
			SetPartialFilterExpression(bson.M{"canonical": true})},
		{Keys: bson.D{{Key: "type", Value: 1}, {Key: "code", Value: 1}}, Options: options.Index().SetName("mapping_type_code_unique").SetUnique(true)},
	}); err != nil {
		return err
	}
	_, err = db.Collection("course").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "proposalId", Value: 1}},
		Options: options.Index().SetName("course_proposal_id_unique").SetUnique(true).
			SetPartialFilterExpression(bson.M{"proposalId": bson.M{"$type": "string", "$gt": ""}}),
	})
	if err != nil {
		return err
	}
	_, err = db.Collection("user").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetName("idx_user_username_unique").SetUnique(true).
			SetCollation(&options.Collation{Locale: "en", Strength: 2}).
			SetPartialFilterExpression(bson.M{"username": bson.M{"$type": "string", "$gt": ""}}),
	})
	return err
}

func mappingNameKey(mappingType model.MappingType, name string) string {
	return fmt.Sprintf("%d:%s", mappingType, name)
}

func mappingCodeKey(mappingType model.MappingType, code int32) string {
	return fmt.Sprintf("%d:%d", mappingType, code)
}

func containsNonPositive(values []int32) bool {
	for _, value := range values {
		if value <= 0 {
			return true
		}
	}
	return false
}

func sortedCodes(values map[int32]struct{}) []int32 {
	result := make([]int32, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func writePlan(path string, value *plan) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func readPlan(path string) (*plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result plan
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "migration failed:", err)
	os.Exit(1)
}

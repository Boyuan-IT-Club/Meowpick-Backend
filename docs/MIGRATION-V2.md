# Backend v2 database migration

This migration makes MongoDB the source of truth for campus, department, and
course-category mappings. Redis is a rebuildable cache. Do not import the old
JSON fixture into production: it is test input, not a production backup.

## What changes

- `mapping._id` becomes an ObjectID.
- `mapping` gains `canonical: bool`. Historical duplicate display names keep all
  `code -> name` aliases; the lowest historical static code is canonical for
  `name -> code`. Newly created mappings are canonical.
- Unique indexes protect `(type, code)` and canonical `(type, name)` records.
- `mapping_counter` stores the next safe sequence per mapping type.
- Legacy comment ObjectID keys are converted to their hexadecimal string form so
  newly returned comment IDs can be used by the like API.
- Courses with code `0` are repaired from their source proposal only when the
  source is unambiguous. Otherwise migration stops and reports the course ID.
- Teacher timestamps affected by the 1970 conversion bug are recovered from a
  valid ObjectID timestamp when possible.
- A partial unique index on `course.proposalId` enforces one proposal to one
  formal course.

The application now uses a MongoDB transaction for proposal approval. Production
MongoDB must therefore be a replica set or sharded deployment; standalone MongoDB
is rejected by the migration preflight.

## Required execution order

1. Stop proposal approval writes or enter a maintenance window.
2. Confirm the application uses the `Eagle233` release being deployed.
3. Back up the current online database.
4. Run dry-run and review the JSON report.
5. Resolve every reported conflict in the online database; never edit the plan to
   hide a conflict.
6. Run dry-run again. Continue only when `conflicts` is empty.
7. Apply that exact plan. The tool verifies that the database snapshot has not
   changed since dry-run.
8. Restart the backend. Startup atomically rebuilds all six Redis Hash keys.
9. Run the validation queries below before reopening writes.

## Backup

Set environment variables in the operator shell; do not place credentials in the
repository.

```bash
export MEOWPICK_MONGO_URI='mongodb://...'
export MEOWPICK_DB='meowpick'
mongodump --uri "$MEOWPICK_MONGO_URI" --db "$MEOWPICK_DB" \
  --archive="meowpick-before-v2-$(date +%Y%m%d-%H%M%S).archive.gz" --gzip
```

Record the archive checksum and store it outside the application host.

## Mandatory dry-run

```bash
go run ./cmd/migrate-v2 \
  --uri "$MEOWPICK_MONGO_URI" \
  --db "$MEOWPICK_DB" \
  --report migration-v2-plan.json
```

Exit code `0` means the report has no conflicts. Exit code `2` means no writes
were made and the `conflicts` array must be resolved. The report contains no
MongoDB credentials.

The tool stops for, among other cases:

- an online dynamic code colliding with a different old static name;
- an online name using a different code from its old static definition;
- a course containing code `0` without an unambiguous source proposal;
- duplicate formal courses for one `proposalId`;
- a comment ObjectID colliding with an existing string ID;
- a standalone MongoDB deployment.

## Conflict investigation templates

Use the IDs and values from the generated report. These commands are read-only:

```javascript
// mongosh, after selecting the production database
db.mapping.find({type: 1, code: 1})
db.course.find({$or: [{department: 0}, {category: 0}, {campuses: 0}]})
db.course.find({proposalId: {$type: "string", $gt: ""}})
  .sort({proposalId: 1})
db.proposal.findOne({_id: "PROPOSAL_ID"}, {course: 1})
```

For a zero-code course without a source proposal, an operator must determine the
correct existing codes from authoritative business data. An explicit repair has
this form; replace all placeholders and retain the old-value predicates:

```javascript
db.course.updateOne(
  {_id: "COURSE_ID", department: 0, category: 0},
  {$set: {department: NumberInt(DEPARTMENT_CODE), category: NumberInt(CATEGORY_CODE)}}
)
```

For an online dynamic/static code collision, do not update only `mapping.code`.
First identify every course and teacher that uses the ambiguous code, determine
which records refer to each meaning, and migrate those references together. If
that decision cannot be made from authoritative data, leave the conflict
unresolved and do not deploy.

After any explicit repair, discard the old plan and run dry-run again.

## Apply

```bash
go run ./cmd/migrate-v2 \
  --uri "$MEOWPICK_MONGO_URI" \
  --db "$MEOWPICK_DB" \
  --apply-plan migration-v2-plan.json
```

Application is refused when the report contains conflicts, has the wrong version
or database, or its snapshot hash differs from the current database. Mapping,
counter, comment-ID, course, and timestamp writes run in one MongoDB transaction.

## Post-migration validation

Run dry-run again. All repair counts and conflicts must be zero:

```bash
go run ./cmd/migrate-v2 \
  --uri "$MEOWPICK_MONGO_URI" \
  --db "$MEOWPICK_DB" \
  --report migration-v2-postcheck.json
```

Useful `mongosh` checks:

```javascript
db.mapping.aggregate([
  {$group: {_id: {type: "$type", code: "$code"}, count: {$sum: 1}}},
  {$match: {count: {$gt: 1}}}
])
db.mapping.countDocuments({canonical: true})
db.mapping_counter.find().sort({_id: 1})
db.course.countDocuments({$or: [{department: 0}, {category: 0}, {campuses: 0}]})
db.comment.countDocuments({_id: {$type: "objectId"}})
```

After backend startup, Redis must contain these six hashes:

```text
mapping:{reference-mappings}:1:name_to_code
mapping:{reference-mappings}:1:code_to_name
mapping:{reference-mappings}:2:name_to_code
mapping:{reference-mappings}:2:code_to_name
mapping:{reference-mappings}:3:name_to_code
mapping:{reference-mappings}:3:code_to_name
```

Validate a sample in both directions with `HGET`, then approve a test proposal and
confirm the course, mappings, proposal status, contribution, and changelog all
commit together.

## Rollback

If validation fails, stop the new backend and restore the pre-migration archive.
Restoring with `--drop` replaces collections and is destructive, so verify the
database name and archive path before running it:

```bash
mongorestore --uri "$MEOWPICK_MONGO_URI" --db "$MEOWPICK_DB" \
  --archive="PATH_TO_VERIFIED_BACKUP.archive.gz" --gzip --drop
```

Flush only the application's Redis database, then restart the previous backend.
Redis mapping data is disposable and will be rebuilt from the restored MongoDB.

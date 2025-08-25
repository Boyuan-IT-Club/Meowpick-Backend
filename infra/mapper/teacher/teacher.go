package teacher

import (
	"time"
)

type Teacher struct {
	ID         string    `bson:"_id,omitempty"  json:"id"`
	Name       string    `bson:"name"           json:"name"`
	Title      string    `bson:"title"          json:"title"`
	Department int32     `bson:"department"     json:"department"`
	CreatedAt  time.Time `bson:"createdAt"      json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt"      json:"updatedAt"`
}

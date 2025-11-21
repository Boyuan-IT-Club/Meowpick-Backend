package searchhistory

import (
	"time"
)

type SearchHistory struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	UserID    string    `bson:"userId"        json:"-"` // 返回给用户时，无需包含用户自己的ID
	Query     string    `bson:"query"         json:"query"`
	CreatedAt time.Time `bson:"createdAt"     json:"createdAt"`
}

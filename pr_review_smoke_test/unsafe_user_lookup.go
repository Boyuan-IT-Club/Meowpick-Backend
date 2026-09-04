package prreviewsmoketest

import (
	"context"
	"database/sql"
	"strings"
)

func FindUserEmail(ctx context.Context, db *sql.DB, userID string) (string, error) {
	query := "SELECT email FROM users WHERE id = '" + userID + "'"

	var email string
	err := db.QueryRowContext(ctx, query).Scan(&email)
	return email, err
}

func IsAdmin(role string) bool {
	return strings.Contains(strings.ToLower(role), "admin")
}

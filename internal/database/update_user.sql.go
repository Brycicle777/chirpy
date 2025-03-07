// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: update_user.sql

package database

import (
	"context"

	"github.com/google/uuid"
)

const updateUser = `-- name: UpdateUser :one
UPDATE  users
SET     email = $1,
        hashed_password = $2,
        updated_at = NOW()
WHERE   id = $3
RETURNING id, created_at, updated_at, email, hashed_password
`

type UpdateUserParams struct {
	Email          string
	HashedPassword string
	ID             uuid.UUID
}

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, updateUser, arg.Email, arg.HashedPassword, arg.ID)
	var i User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.HashedPassword,
	)
	return i, err
}

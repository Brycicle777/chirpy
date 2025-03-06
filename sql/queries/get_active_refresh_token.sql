-- name: GetActiveRefreshToken :one
SELECT
          token
          ,created_at
          ,updated_at
          ,user_id
          ,expires_at
          ,revoked_at
FROM      refresh_tokens
WHERE     token = $1
      AND NOW() < expires_at
      AND revoked_at IS NULL;
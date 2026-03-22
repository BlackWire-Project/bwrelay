-- ============ USERS ============

-- name: CreateUser :one
INSERT INTO users (username, identity_key, signed_prekey, signed_prekey_signature, inbox_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, username, inbox_id, created_at;

-- name: GetUserByUsername :one
SELECT id, username, identity_key, signed_prekey, signed_prekey_signature, inbox_id, created_at
FROM users
WHERE username = $1;

-- name: GetUserByInboxID :one
SELECT id, username, identity_key, signed_prekey, signed_prekey_signature, inbox_id, created_at
FROM users
WHERE inbox_id = $1;

-- name: UserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = $1) AS exists;

-- ============ ONE-TIME PREKEYS ============

-- name: CreatePrekey :one
INSERT INTO one_time_prekeys (user_id, prekey)
VALUES ($1, $2)
RETURNING id;

-- name: GetPrekeyByUsername :one
SELECT p.id, p.user_id, p.prekey, p.created_at
FROM one_time_prekeys p
INNER JOIN users u ON u.id = p.user_id
WHERE u.username = $1
ORDER BY p.created_at ASC
LIMIT 1;

-- name: GetPrekeysCountByUsername :one
SELECT COUNT(*) AS count
FROM one_time_prekeys p
INNER JOIN users u ON u.id = p.user_id
WHERE u.username = $1;

-- ============ MESSAGES ============

-- name: CreateMessage :one
INSERT INTO messages (inbox_id, kind, header, ciphertext, used_prekey_id, client_message_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at, expires_at;

-- name: GetMessagesByInboxID :many
SELECT id, inbox_id, kind, header, ciphertext, used_prekey_id, client_message_id, created_at, expires_at
FROM messages
WHERE inbox_id = $1
  AND expires_at > NOW()
ORDER BY created_at ASC
LIMIT $2;

-- name: DeleteExpiredMessages :execrows
DELETE FROM messages
WHERE expires_at <= NOW();

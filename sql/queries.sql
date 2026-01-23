-- ============ USERS ============

-- name: CreateUser :one
INSERT INTO users (username, identity_key, signed_prekey, signed_prekey_signature)
VALUES ($1, $2, $3, $4)
RETURNING id, username, created_at;

-- name: GetUserByUsername :one
SELECT id, username, identity_key, signed_prekey, signed_prekey_signature, created_at
FROM users
WHERE username = $1;

-- name: UserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = $1) AS exists;

-- ============ ONE-TIME PREKEYS ============

-- name: CreatePrekey :one
INSERT INTO one_time_prekeys (username, prekey)
VALUES ($1, $2)
RETURNING id;

-- name: GetPrekey :one
SELECT id, username, prekey, created_at
FROM one_time_prekeys
WHERE username = $1
ORDER BY created_at ASC
LIMIT 1;

-- name: DeletePrekeyByID :one
DELETE FROM one_time_prekeys
WHERE id = $1
RETURNING id;

-- name: GetPrekeysCount :one
SELECT COUNT(*) AS count
FROM one_time_prekeys
WHERE username = $1;

-- ============ MESSAGES ============

-- name: CreateMessage :one
INSERT INTO messages (recipient, payload, dh_public, message_number, previous_chain_length)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at;

-- name: GetMessageByID :one
SELECT id, recipient, payload, dh_public, message_number, previous_chain_length, created_at
FROM messages
WHERE id = $1;

-- name: GetMessagesByRecipient :many
SELECT id, recipient, payload, dh_public, message_number, previous_chain_length, created_at
FROM messages
WHERE recipient = $1
ORDER BY created_at ASC;

-- name: GetMessagesByRecipientAfter :many
SELECT id, recipient, payload, dh_public, message_number, previous_chain_length, created_at
FROM messages
WHERE recipient = $1 AND created_at > $2
ORDER BY created_at ASC
LIMIT $3;

-- name: DeleteExpiredMessages :execrows
DELETE FROM messages
WHERE created_at < NOW() - INTERVAL '30 days';

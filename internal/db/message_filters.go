package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrInvalidMessageCursor = errors.New("invalid message cursor")

type ListMessagesFilter struct {
	InboxID       string
	Limit         int
	AfterID       string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

const listMessagesQuery = `
SELECT id, inbox_id, kind, header, ciphertext, used_prekey_id, client_message_id, created_at, expires_at
FROM messages
WHERE inbox_id = $1
  AND expires_at > NOW()
  AND ($2::timestamp IS NULL OR created_at >= $2)
  AND ($3::timestamp IS NULL OR created_at <= $3)
  AND (
    $4::timestamp IS NULL OR $5::uuid IS NULL
    OR (created_at, id) > ($4, $5)
  )
ORDER BY created_at ASC, id ASC
LIMIT $6
`

const getMessageCursorQuery = `
SELECT id, created_at
FROM messages
WHERE inbox_id = $1
  AND id = $2
LIMIT 1
`

const getMessageByIDQuery = `
SELECT id, inbox_id, kind, header, ciphertext, used_prekey_id, client_message_id, created_at, expires_at
FROM messages
WHERE inbox_id = $1
  AND id = $2
  AND expires_at > NOW()
LIMIT 1
`

func (q *Queries) ListMessages(ctx context.Context, filter ListMessagesFilter) ([]Message, error) {
	if filter.Limit <= 0 {
		filter.Limit = 1
	}

	var afterCreatedAt pgtype.Timestamp
	var afterUUID pgtype.UUID

	if filter.AfterID != "" {
		parsedID, err := parseUUID(filter.AfterID)
		if err != nil {
			return nil, ErrInvalidMessageCursor
		}

		row := q.db.QueryRow(ctx, getMessageCursorQuery, filter.InboxID, parsedID)
		if err := row.Scan(&afterUUID, &afterCreatedAt); err != nil {
			return nil, ErrInvalidMessageCursor
		}
	}

	rows, err := q.db.Query(
		ctx,
		listMessagesQuery,
		filter.InboxID,
		toNullableTimestamp(filter.CreatedAfter),
		toNullableTimestamp(filter.CreatedBefore),
		afterCreatedAt,
		afterUUID,
		int32(filter.Limit),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Message{}
	for rows.Next() {
		var item Message
		if err := rows.Scan(
			&item.ID,
			&item.InboxID,
			&item.Kind,
			&item.Header,
			&item.Ciphertext,
			&item.UsedPrekeyID,
			&item.ClientMessageID,
			&item.CreatedAt,
			&item.ExpiresAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (q *Queries) GetMessageByID(ctx context.Context, inboxID string, messageID string) (Message, error) {
	var message Message

	parsedID, err := parseUUID(messageID)
	if err != nil {
		return message, ErrInvalidMessageCursor
	}

	row := q.db.QueryRow(ctx, getMessageByIDQuery, inboxID, parsedID)
	if err := row.Scan(
		&message.ID,
		&message.InboxID,
		&message.Kind,
		&message.Header,
		&message.Ciphertext,
		&message.UsedPrekeyID,
		&message.ClientMessageID,
		&message.CreatedAt,
		&message.ExpiresAt,
	); err != nil {
		return message, err
	}

	return message, nil
}

func toNullableTimestamp(value *time.Time) pgtype.Timestamp {
	if value == nil {
		return pgtype.Timestamp{}
	}

	return pgtype.Timestamp{
		Time:  *value,
		Valid: true,
	}
}

func parseUUID(value string) (pgtype.UUID, error) {
	var uuid pgtype.UUID

	clean := ""
	for _, char := range value {
		if char != '-' {
			clean += string(char)
		}
	}

	if len(clean) != 32 {
		return uuid, fmt.Errorf("invalid uuid length")
	}

	var bytes [16]byte
	for i := 0; i < 16; i++ {
		var parsed byte
		if _, err := fmt.Sscanf(clean[i*2:i*2+2], "%02x", &parsed); err != nil {
			return uuid, err
		}
		bytes[i] = parsed
	}

	uuid.Bytes = bytes
	uuid.Valid = true
	return uuid, nil
}

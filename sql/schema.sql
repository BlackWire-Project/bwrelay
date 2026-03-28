-- Users table
CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    identity_key TEXT UNIQUE NOT NULL,
    signed_prekey TEXT UNIQUE NOT NULL,
    signed_prekey_signature TEXT NOT NULL,
    inbox_id VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- One-time prekeys table
CREATE TABLE IF NOT EXISTS one_time_prekeys(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    prekey TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Messages table
CREATE TABLE IF NOT EXISTS messages(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inbox_id VARCHAR(255) NOT NULL REFERENCES users(inbox_id) ON DELETE CASCADE,
    kind VARCHAR(32) NOT NULL,
    header TEXT NOT NULL,
    ciphertext TEXT NOT NULL,
    used_prekey_id UUID NULL,
    client_message_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days'),
    UNIQUE (inbox_id, client_message_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_inbox_id
ON users(inbox_id);

CREATE INDEX IF NOT EXISTS idx_prekeys_user_id
ON one_time_prekeys(user_id);

CREATE INDEX IF NOT EXISTS idx_messages_inbox_created
ON messages(inbox_id, created_at);

CREATE INDEX IF NOT EXISTS idx_messages_pending
ON messages(inbox_id, expires_at, created_at);

CREATE INDEX IF NOT EXISTS idx_messages_inbox_created_id
ON messages(inbox_id, created_at, id);

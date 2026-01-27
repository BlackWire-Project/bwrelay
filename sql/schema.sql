-- Users table
CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    identity_key TEXT UNIQUE NOT NULL,
    signed_prekey TEXT UNIQUE NOT NULL,
    signed_prekey_signature TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- One-time prekeys table
CREATE TABLE IF NOT EXISTS one_time_prekeys(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    prekey TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- Messages table
CREATE TABLE IF NOT EXISTS messages(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient VARCHAR(255) NOT NULL REFERENCES users(username),
    payload TEXT NOT NULL,
    dh_public TEXT NOT NULL,
    message_number INTEGER NOT NULL,
    previous_chain_length INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- Indexes
CREATE INDEX IF NOT EXISTS idx_prekeys_username
ON one_time_prekeys(username);
CREATE INDEX IF NOT EXISTS idx_messages_recipient
ON messages(recipient);
CREATE INDEX IF NOT EXISTS idx_messages_created_at
ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_recipient_created
ON messages(
    recipient,
    created_at
);

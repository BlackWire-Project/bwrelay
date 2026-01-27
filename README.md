# bwrelay

Dumb relay server for end-to-end encrypted messaging. Part of the BlackWire project.

## What is this?

A relay server that stores and forwards encrypted messages. It never sees plaintext - all encryption happens client-side using X3DH + Double Ratchet (Signal Protocol).

The relay is intentionally "dumb":
- No authentication
- No message inspection
- No federation
- Just stores blobs and notifies recipients

Why?

Because we must not trust even who hosts the relay, **trust the cryptography**, so you can make a proper use even don't knowing who is hosting the relay, even no authentication, it does not matter, cryptography is the only way to communicate safely to bypass even who is always watching you.

## Quick Start

```bash
# 1. Start PostgreSQL
docker compose up -d

# 2. Apply schema
docker exec -i bwrelay psql -U dev -d bwrelay < sql/schema.sql

# 3. Run server
go run ./cmd/bwrelay/main.go
```

Server runs on `http://localhost:8080`

## Configuration

Environment variables (or `.env` file):

```
DATABASE_URL=postgres://dev:root@localhost:5432/bwrelay?sslmode=disable
PORT=8080
```

## Database Schema

```mermaid
erDiagram
    users {
        uuid id PK
        varchar username UK
        text identity_key UK
        text signed_prekey UK
        text signed_prekey_signature
        timestamp created_at
    }

    one_time_prekeys {
        uuid id PK
        varchar username FK
        text prekey UK
        timestamp created_at
    }

    messages {
        uuid id PK
        varchar recipient FK
        text payload
        text dh_public
        int message_number
        int previous_chain_length
        timestamp created_at
    }

    users ||--o{ one_time_prekeys : has
    users ||--o{ messages : receives
```

## API

### Users

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/users` | Register user with keys |
| GET | `/users/:username` | Get user's public keys |
| GET | `/users/:username/prekey` | Get one-time prekey (for X3DH) |
| POST | `/users/:username/prekeys` | Add more one-time prekeys |

### Messages

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/messages` | Send encrypted message |
| GET | `/messages/:id` | Get message by ID |
| GET | `/messages?recipient=X` | List messages for recipient |

### WebSocket

| Endpoint | Description |
|----------|-------------|
| GET `/ws` | Real-time notifications |

**WebSocket Protocol:**

```json
// Client -> Server: Subscribe
{"type": "subscribe", "username": "bob"}

// Server -> Client: New message notification
{"type": "new_message", "id": "uuid"}
```

## Message Flow

```mermaid
sequenceDiagram
    participant A as Alice
    participant R as Relay
    participant B as Bob

    Note over A,B: First message (X3DH)

    A->>R: GET /users/bob
    R-->>A: identity_key, signed_prekey
    A->>R: GET /users/bob/prekey
    R-->>A: prekey_id, one_time_prekey

    Note over A: Compute shared secret (X3DH)
    Note over A: Encrypt message

    A->>R: POST /messages {recipient, payload, prekey_id}
    R->>R: Delete used prekey
    R->>B: WS: new_message notification

    B->>R: GET /messages/:id
    R-->>B: encrypted payload

    Note over B: Compute shared secret (X3DH)
    Note over B: Decrypt message

    Note over A,B: Subsequent messages (Double Ratchet)

    A->>R: POST /messages {recipient, payload, message_number: 1}
    R->>B: WS: new_message notification
```

## Key Concepts

**X3DH (Extended Triple Diffie-Hellman)**
- Used for first message only
- Allows messaging offline users
- `prekey_id` required when `message_number == 0`

**Double Ratchet**
- Used for all subsequent messages
- Forward secrecy (compromise one key ≠ compromise all)
- `message_number` tracks position in ratchet

**Zero Trust**
- Relay never sees plaintext
- Works over HTTP or HTTPS (E2E protects either way)
- Attacker with full relay access sees only encrypted blobs

## Project Structure

```
bwrelay/
├── cmd/bwrelay/main.go    # Entry point
├── internal/
│   ├── config/            # Environment config
│   ├── db/                # Generated sqlc code
│   ├── handler/           # HTTP handlers
│   └── ws/                # WebSocket hub
└── sql/
    ├── schema.sql         # Database schema
    └── queries.sql        # SQL queries (sqlc)
```

## License

MIT

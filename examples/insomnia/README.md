# Insomnia Fixtures

This folder contains an Insomnia export for exercising the current `bwrelay` API with two users.

## Files

- `bwrelay-two-users.insomnia.json`: importable Insomnia workspace

## Expected Local Setup

- relay API running on `http://127.0.0.1:8080`
- PostgreSQL already running and schema applied
- clean database recommended before the first import run

## Flow Included

1. Register Alice with public bundle, `inbox_id`, and one-time prekeys
2. Register Bob with public bundle, `inbox_id`, and one-time prekeys
3. Fetch Bob's bundle
4. Alice sends a `prekey_message` to Bob
5. Bob polls his inbox
6. Fetch Alice's bundle
7. Bob sends a `ratchet_message` to Alice
8. Alice polls her inbox

## Import

In Insomnia:

1. `Create` -> `Import`
2. Select `From File`
3. Choose `examples/insomnia/bwrelay-two-users.insomnia.json`

## Variables You May Want To Change

The workspace environment contains:

- `base_url`
- `alice_username`
- `bob_username`
- `alice_inbox_id`
- `bob_inbox_id`
- `alice_bundle_prekey_id`
- `bob_bundle_prekey_id`

Notes:

- Registration requests use fixed inbox IDs suitable for local testing.
- After `GET Bundle - Bob`, copy `prekey_id` into `bob_bundle_prekey_id` if needed.
- After `GET Bundle - Alice`, copy `prekey_id` into `alice_bundle_prekey_id` if needed.

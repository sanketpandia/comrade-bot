# Encryption at rest

How Politburo should protect stored credentials on the single production VPS,
without encrypting live flight cache, livery catalogs, or ordinary VA settings.

Two different losses matter, and one control does not cover both:

| Loss | What actually leaves the machine |
|---|---|
| Offline copy of the data disk | Hetzner volume snapshot, or the Postgres and Redis files on the `local-path` PVCs |
| Logical copy of a store | `pg_dump`, a `psql` session as `ieuser`, a Redis AOF/RDB copy, or `kubectl get secret` |

Disk encryption stops the first. It does nothing for the second, because the
dump is taken through an already-unlocked database. Sealing individual
credentials stops the second for those values. It does nothing for a stolen
volume full of plaintext flights, sessions, and identities.

Use both. Keep each one small.

## What production stores today

From `infra/prod/k8s/base/data/` and the rewrite migrations.

Postgres 15 and Redis 7 are single-replica StatefulSets in `ie-data`. Each
has its own PVC (`local-path`, 20Gi and 5Gi). Redis runs with
`--appendonly yes` and `requirepass`. Neither service has encryption at rest.
Community Postgres has no transparent data encryption, and neither does
Redis. Kubernetes Secrets for the database password, Redis password, Infinite
Flight key, OAuth client secret, signed-link secret, and Discord bot token
are base64 in the k3s datastore. Base64 is encoding.

The database password and API keys the process needs to *start* stay in
those Secrets (or `*_FILE` mounts). This document is about data the app
*writes* after it is running.

### Postgres

| Place | What is sensitive | Notes |
|---|---|---|
| `api_keys.id` | The row id **is** the bearer token | `INSERT INTO api_keys (id, status)`. Lookup is `WHERE id = $1`. A dump is a list of working keys. |
| `va_configs.config` where `config_type = 'WEBHOOKS'` | Discord webhook URL | URL shape is `https://discord.com/api/webhooks/{id}/{token}`. The token is the secret. `frequency` and `filters` are not. |
| `va_configs.config` where `config_type = 'BASIC'` | Nothing | Callsign affix and preferred server names. |
| `users`, `va_user_roles`, `banned_discord_ids`, `platform_reports` | Identifiers | Discord ids, IFC usernames, callsigns, report notes. Not credentials. |
| `user_deletion_archives.payload` | Copy of the user and memberships | Operator archive. Identifiers, not credentials. |
| `contact_submissions` | IFC username and free-text message | A user could paste a secret. We do not treat the column as a credential store. |
| `va_webhook_runs` | Nothing by design | Outcome and timing only. Discord client errors already omit the URL. |
| `game_aircraft_liveries` | Nothing | Public catalog. |

`va_configs` is one jsonb document per `(va_id, config_type)`. Jobs and the
API load the whole document and parse it in Go. Nothing in SQL extracts
`config->'active_flights'->>'url'`. Sealing one field inside the document
does not break a query.

Planned tables, not migrated yet, already assume a sealed credential:

- Airtable personal access token on `va_data_provider_configs`
  ([airtable-datasource.md](airtable-datasource.md)). The base id and the
  field map stay readable. The token does not go in Redis.
- Live connection tokens, when they become durable
  ([live-mode.md](live-mode.md)). Today they exist only in Redis.

### Redis

AOF persists every key below onto the PVC. TTLs limit how long a value lives.
They do not keep it off disk.

| Key | Lifetime | Secret? |
|---|---|---|
| `auth:if_oauth_user:{userId}` | 7 days (browser session TTL) | Yes. `access_token` and `refresh_token` in JSON. This is the Live connection until it moves to Postgres. |
| `auth:if_oauth:{state}` | 10 minutes | Yes. PKCE `code_verifier`. |
| `auth:api_key:{raw uuid}` | 1 minute | The **key name** is the API key. The value is only `{active, found}`. |
| `auth:login_ticket:{id}` | 10 minutes | The URL token is already AES-GCM with `SIGNED_LINK_SECRET`. The Redis value is identity claims. The key name is a random id, not the URL token. |
| `session:{sessionId}` | 7 days | The key name is the cookie. The value is user id, Discord id, VA, role. Possession of the key name is the session. |
| `game:sessions:*`, `game:flights:*`, `game:livery:*` | Minutes | Live Infinite Flight cache. Large, rewritten constantly, not a credential. |
| `provider:airtable:{vaId}:preview:*` (planned) | 15 minutes | One sample row and column types. The Airtable token must not be stored here. |

Future VA config and future Redis features will add more of the same mix:
mostly structure and cache, occasionally a token inside a JSON document or
under a new prefix.

## Whole store vs credentials only

| Approach | Offline disk snapshot | `pg_dump` / Redis copy / `psql` | App overhead | Fits this host |
|---|---|---|---|---|
| LUKS on the data volume | Postgres files, Redis AOF, Loki, Grafana on that volume are ciphertext | No. Dumps are plaintext | None in Politburo. One unlock, then a copy of existing PVCs onto the new device | Yes. This is the "encrypt the whole thing" that is actually available |
| Encrypt every Postgres row or the whole cluster with `pgcrypto` | Only if the key is not next to the data | Yes, but every read pays for it, jsonb operators die, and liveries and flights get wrapped for no reason | High, and ongoing | No |
| Redis built-in encryption at rest | Redis does not have it. Redis Enterprise does | — | A second product | No |
| Seal every Redis value | Same as app sealing | Yes | AES-GCM on flight snapshots every minute, for data that is not secret | No |
| Seal credential fields and credential Redis prefixes | No | Yes, for those values | AES-GCM on short strings, on the write and read path that already parses JSON | Yes |
| Hash API keys | No | A dump no longer contains a usable key | One SHA-256 on each `X-API-Key` check. The 1-minute cache stays | Yes. Encryption is the wrong tool: we only ever test equality |

A sealed column still decrypts for anyone who has both the dump and the data
key. The data key will live in a Kubernetes Secret. Turn on k3s secrets
encryption so that Secret is not recoverable from a copy of the k3s sqlite
file alone. That is a third, small control, aimed at the cluster datastore
on the root disk rather than at Postgres.

None of this helps once the Politburo process is compromised. The process
holds the data key, the database password, and the Redis password. Disk
encryption also does not help while the volume is unlocked, which is the
whole time the VPS is up. The snapshot case is a copy taken of the block
device, or a backup taken off the machine.

## Decision

1. **LUKS the data volume** that `local-path` uses (`/mnt/HC_Volume_*` in
   [infra/prod/k8s/README.md](../../../infra/prod/k8s/README.md)). Postgres,
   Redis AOF, and the observability PVCs on that disk are covered together.
   The passphrase or key file stays off that volume. A key file sitting on
   the same filesystem as the ciphertext does not survive a full VM image.
2. **One application sealer** for credentials inside JSON and Redis values.
   AES-256-GCM, same construction already used for signed-link ticket ids
   (`internal/access/auth/ticket.go`): random nonce, tag, no extra service
   and no Vault. A new key, `DATA_ENCRYPTION_KEY` / `DATA_ENCRYPTION_KEY_FILE`,
   loaded through the existing `secret()` helper. Do not reuse
   `SIGNED_LINK_SECRET`. Rotating one must not invalidate the other.
3. **Hash API keys.** Store `token_hash`, look up by the hash, stop putting
   the raw token in `api_keys.id` and in the Redis key name.
4. **Enable k3s secrets encryption** so the data key and the other process
   secrets are encrypted in the k3s datastore.

Leave BASIC configs, memberships, archives, contact messages, webhook run
rows, liveries, and the entire `game:*` cache in the clear. Disk encryption
and ordinary access control (Redis password, ClusterIP only, no public
Postgres) are the controls for that data.

## Sealer

Stdlib only: `crypto/aes` and GCM. Ciphertext is a string the rest of the
system can store in `text`, `jsonb`, or a Redis value:

```text
enc:v1:<key-id>:<base64 nonce||ciphertext||tag>
```

`v1` is the format. `<key-id>` is which key wrapped it, so a later key can
open old values. Non-secret code never inspects the payload.

Rules:

- Seal on the way into the store, open on the way out, at one boundary per
  store. `va_configs` repository does it for documents. The Redis store does
  it for listed prefixes. Handlers, webhook jobs, and OAuth code keep using
  plaintext.
- Validate before sealing. A Discord webhook URL is checked with
  `ValidDiscordWebhookURL` on the raw string, then the `url` field is replaced
  with the envelope. Frequency and filters are written unchanged.
- Opening a value that is not an envelope returns it unchanged during the
  migration window. After the rewrite job, a credential field that is still
  plaintext is a bug.
- Production refuses to start without `DATA_ENCRYPTION_KEY` once sealing is
  on. Local uses a documented dev key, the same pattern as
  `SIGNED_LINK_SECRET`. The dev key is public and must not be the production
  key.

### JSON documents

Seal fields, not the document. A WEBHOOKS row stays a normal object:

```json
{
  "active_flights": {
    "url": "enc:v1:1:…",
    "frequency": "30m",
    "filters": { "vaCode": "AFK" }
  }
}
```

A registry names the secret paths per `config_type`:

| `config_type` | Sealed paths | Left as plaintext |
|---|---|---|
| `WEBHOOKS` | `url` on every feature object, including feature keys this build does not understand yet | `frequency`, `filters`, feature names |
| `BASIC` | none | the whole document |
| Future provider row (`va_data_provider_configs`) | the access token column or token field | base id, table names, field map |
| Future Live connection row | `access_token`, `refresh_token` | user id, scopes, expiry, org ids |
| Anything else | none, until someone adds a path | the whole document |

New config types default to "no sealed paths." Adding a credential means
adding a path to the registry in the same change that introduces the field.
Unknown feature objects under `WEBHOOKS` still seal `url` when the property
is a string, so a new webhook feature does not get a plaintext token just
because the binary is older than the feature.

Do not encrypt the whole `config` blob. Operators and future SQL will want
to see frequencies and flags. Wrapping the blob would also re-encrypt
harmless fields every time a secret is rotated.

`user_deletion_archives.payload` and `contact_submissions.message` stay
plaintext. They are read by operators as records, and they are not a
credential API. Disk encryption covers the offline copy.

### Redis prefixes

A prefix list next to `internal/cache/keys.go` decides which values are
sealed. The key name stays readable so TTL, `SCAN`, and delete-by-id keep
working.

| Prefix | Value |
|---|---|
| `auth:if_oauth_user:` | Seal the JSON |
| `auth:if_oauth:` | Seal the JSON |
| `auth:api_key:` | Stop using the raw token as the key. Key becomes `auth:api_key:{hex hash}`. Value stays `{active, found}` and is not sealed |
| `session:`, `auth:login_ticket:`, `game:*`, provider previews | Plaintext |

A new Redis feature that stores a token, refresh token, client secret, or
webhook URL adds its prefix to the seal list before it ships. Cache of game
state, ids, and short-lived previews stays off the list. The default for a
new prefix is plaintext, so sealing is a deliberate line in the key file
rather than a surprise tax on flight snapshots.

When Live connections move to Postgres, Postgres is the copy that survives
restart. Redis may cache the sealed blob or a short-lived access token under
the sealed prefix. Disconnect and user deletion delete the Postgres row and
the Redis key in the same step. There should not be a plaintext token in AOF
with a longer life than the session that created it.

### API keys

Hashing, not sealing. The server never needs the token back, only a yes/no.

- New column `token_hash` (`bytea`, unique), SHA-256 of the presented token.
- `id` becomes an ordinary row uuid. Clients keep sending the token they were
  shown once. Lookup hashes it and selects on `token_hash`.
- Redis cache key uses the same hash. `MONITOR`, slowlog, and AOF then show
  a hash, not a key that can be replayed as `X-API-Key`.
- Existing rows cannot be migrated in place: the only stored form of the
  token is the uuid, and leaving that uuid in the table keeps the leak.
  Reissue the bot `API_KEY` and any operator keys, insert the hash, drop the
  old row.

Until that reissue, the live protection for `api_keys` is LUKS plus not
shipping dumps around.

### Key rotation

Ship a second key as decrypt-only (`DATA_ENCRYPTION_KEY_PREVIOUS`, with its
key id). New writes use the current key. A one-shot rewrite walks
`va_configs` and any later credential tables and writes them back through
the repository, which seals with the current key. Redis entries expire on
their own; keep the previous key for at least seven days (the OAuth session
TTL) after a rotation. API key hashes do not rotate with this key. Losing
the data key means re-entering webhook URLs and reconnecting Infinite Flight.
There is no escrow.

The rewrite is a small Politburo command, not a SQL expression. The key must
not appear in `psql` history or in a migration file.

### Backups

`pg_dump` of sealed fields is safe to store away from the server. Restore
needs the same data key, which is not inside the dump. The rest of the dump
(users, archives, contact messages, sessions are not in Postgres) is still
personal data. Keep dump files on the encrypted volume or wrap the file.
The current runbook writes `backup.sql` to whatever directory the operator
redirects to; that directory needs to be on the encrypted volume.

Do not copy Redis AOF off the box as a backup of credentials. After sealing,
session key names in that file are still bearer cookies.

### Logs

Webhook run rows and the Discord client already avoid storing the URL.
Keep that rule for the sealer: log `va_id`, `config_type`, and key id, never
the opened value and never the envelope. A sealed string in Loki is still a
secret for anyone who later gets the data key.

## Rollout

Order is fixed where a later step assumes an earlier one. Disk encryption and
k3s secrets encryption can proceed in parallel with the code.

1. **LUKS the data volume.** Maintenance window. Snapshot, create the
   encrypted device, copy the PVC directories, point `local-path` at the
   opened mapper device, confirm Postgres and Redis come up from the copied
   data. Document the unlock so a reboot does not start pods against an
   empty directory.
2. **k3s `--secrets-encryption`.** New writes are encrypted. Existing Secrets
   need a rewrite (`kubectl get secret -o yaml` and apply, or recreate from
   the env files) to be stored under the provider. Confirm the encryption
   config key is not on the data volume.
3. **Sealer and `DATA_ENCRYPTION_KEY`.** Local dev key. Production fails
   closed when the key is absent. No behavior change until a path or prefix
   is registered.
4. **WEBHOOKS `url`.** Register the path. Deploy so new saves seal. Run the
   rewrite command so existing rows seal. Reads accept both until that
   command finishes.
5. **OAuth Redis prefixes.** Deploy the prefix list. Users reconnect, or
   existing plaintext values keep working until their 7-day TTL because open
   accepts plaintext. After one TTL, plaintext under those prefixes is an
   error log, not a silent accept.
6. **API key reissue.** Add `token_hash`, write the bot's new key, switch
   `API_KEY`, delete the old row, then change the cache key to the hash.
7. **Later tables use the sealer in their first migration.** Airtable token
   and Live connection tokens do not land as plaintext columns "for now."

## What we are accepting

- A live root compromise reads memory and the mounted data key.
- Identifiers, contact messages, and deletion archives are readable by
  anyone with the database password and an unlocked disk.
- `session:{id}` in AOF is a session cookie until it expires. Sealing the
  JSON under it would not change that.
- Flight cache on disk is readable from an unlocked volume. That matches the
  product rule that Live data is a short-lived cache, not a warehouse, and
  it is the bulk of Redis writes. Encrypting it would spend the overhead in
  the one place that does not hold a credential.

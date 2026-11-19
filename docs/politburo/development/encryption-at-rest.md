# Shared encryption at rest for Politburo

**Status.** Implemented in `services/politburo/internal/secretstore`. The shared-key, malformed-nonce, and cursor-parsing findings below are fixed there. Cluster-admin kubeconfig on the deploy runner, plaintext OAuth sessions in Redis, and the key-recovery owner are still open and need operator action.

**Proposed decision.** Replace the connection-specific cipher and `DATA_CONNECTIONS_ENCRYPTION_KEY` with one Politburo encryption module configured by `DATA_ENCRYPTION_KEY`. Use AES-256-GCM with a randomly generated 256-bit root key per environment. The current cipher already uses AES-256. The improvement is keeping its key separate from cursor signing. Its first consumer is the Airtable personal access token (PAT) added in commit `7ea0929`. The feature is unused and in development, so the cutover can use a fresh key and envelope format. An old environment-variable alias, a legacy reader and a data migration would add no value for this cutover.

## What the recent change actually stores

| Path | Behavior today | Requirement for the replacement |
| --- | --- | --- |
| Configuration and startup | The config loader reads the key from the environment or its `_FILE` counterpart and decodes 32 bytes of standard base64. An absent key leaves the connections service disabled. A malformed key fails startup. | Preserve both failure modes. Other Politburo functions can run without an at-rest key. |
| Create and replace | Create validates the PAT against Airtable, assigns a connection ID and seals the PAT. Update validates a replacement against the configured bases and seals it under the existing connection ID. A name-only update retains the envelope. | Keep validation and conditional writes in the connections service. A failed validation, seal or revision check must leave the stored credential intact. |
| Persistence and display | The credential sits at `connections[].airtable.credential` in the `DATA_CONNECTIONS` JSON block of `public.va_configs.config`. The repository uses the document revision for conditional writes. The admin projection exposes `hasCredential` and generic config routes restrict this block. | Encrypt the credential field. Keep the revision, IDs, mappings and edit metadata available to their existing readers. Listing the admin view needs no decryption. |
| Live reads | `tokenForConnection` opens the selected credential for provider operations, including schema inspection, preview and mapped reads. | Open at the point of use. A corrupt or misplaced credential must stop the operation before an authenticated provider request. |
| Pagination | `CredentialCipher.SigningKey()` hands the AES key to the connections service. List uses that same byte slice for HMAC-SHA256 over the pagination payload. | Give the cursor signer separate key material and preserve VA, entity, filters and revision checks. |

The current envelope contains a version, a random nonce and authenticated ciphertext. Its associated data binds the version, VA ID and connection ID. One failure path deserves a regression test during extraction. The current reader checks nonce encoding but passes any decoded length to GCM, which can panic on a malformed stored nonce. [Current cipher](../../../services/politburo/internal/community/virtualairlines/connections/crypto.go) · [Connection lifecycle](../../../services/politburo/internal/community/virtualairlines/connections/service.go) · [Credential replacement](../../../services/politburo/internal/community/virtualairlines/connections/admin.go) · [Cursor format](../../../services/politburo/internal/community/virtualairlines/connections/cursor.go)

## What cursor signing means

A cursor is a bookmark for the next page. For example, after listing the first page of pilots for a virtual airline (VA), Politburo's list service can return a cursor containing Airtable's next offset, the VA ID, entity, filters and config revision. It attaches an HMAC-SHA256 tag computed with a server-held key. On the next request, the service recomputes the tag, compares it with `hmac.Equal`, and checks that the cursor belongs to the current query before using the offset.

The tag lets Politburo detect edits. Anyone holding the cursor can decode its base64 payload and read its fields. Keep credentials out of it, avoid logging whole cursors, and authorize every page request against current VA access. A valid cursor can be presented again while its query context remains valid. [Cursor implementation](../../../services/politburo/internal/community/virtualairlines/connections/cursor.go)

## Security review

Severity reflects the impact if deployed with real credentials. The evidence below comes from the repository. Production permissions, stored credentials and backup controls require operator verification.

| Priority and severity | Evidence | Potential impact | Practical fix and verification |
| --- | --- | --- | --- |
| 1 · High · k3s admin access | The bootstrap runbook requests kubeconfig mode `644` and copies the k3s admin kubeconfig to the CI runner. The proposed root would live in `politburo-env`. | A principal with that kubeconfig has cluster-wide access, including the root key and database credentials. | Correct the runbook before provisioning real PATs. Keep the admin kubeconfig root-only at `0600`, replace the runner copy with a dedicated deploy identity, and scope each job to necessary workloads and namespaces. Verify effective file access and runner RBAC, including workload creation and modification, which can expose mounted secrets indirectly. Check datastore encryption and backup access separately. [Bootstrap](../../../infra/prod/k8s/README.md) · [K3s admin identity](https://docs.k3s.io/cluster-access) · [Kubernetes Secret controls](https://kubernetes.io/docs/concepts/security/secrets-good-practices/) |
| 2 · High if active · plaintext OAuth credentials | The OAuth callback and VA token provider store access and refresh tokens as JSON in Redis. The proposed Airtable cutover covers Postgres only. | Redis readers or backups containing live sessions could expose reusable provider credentials, including refresh tokens. | Verify whether OAuth is enabled and whether snapshots or replicas retain live sessions. If active, prioritize the encrypted token-session store described below and plan revocation or reconnect for exposed sessions. [Callback](../../../services/politburo/internal/transport/http/api/oauth/handler.go) · [Token provider](../../../services/politburo/internal/livegame/valive/token_provider.go) |
| 3 · Medium · shared raw key | The current cipher returns its AES key to the cursor signer. | Disclosure in either component compromises both credential confidentiality and cursor integrity for that environment. | Implement the proposed labeled subkeys, remove the key accessor and test that the signer and cipher get different keys. Restrict root-key access to initialization. [Current cipher](../../../services/politburo/internal/community/virtualairlines/connections/crypto.go) |
| 4 · Medium · recovery and revocation | The envelope identifies a format version. The rollout mentions future rotation but leaves the production owner and recovery procedure open. | A lost root makes every stored PAT unreadable. A stolen PAT can remain usable at Airtable after a local key change. | Before production, assign an operator and rehearse a restore. For the first release, coordinated re-entry is sufficient. For suspected exposure, revoke affected PATs at Airtable, replace the root and re-enrol with fresh PATs restricted to schema and record reads on the intended bases. Verify actual scopes with the credential owner. [OWASP key lifecycle guidance](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) |
| 5 · Medium · malformed credential handling | The existing decryptor passes a decoded nonce to GCM without checking its length. | A malformed stored credential can panic on repeated reads, interrupting requests for that connection. | Use the proposed random-nonce AEAD and require malformed input to return an error. Exercise truncated and corrupt records through the service, and verify zero provider calls. [Current decryptor](../../../services/politburo/internal/community/virtualairlines/connections/crypto.go) |
| 6 · Low · cursor availability | The decoder scans for a delimiter inside a binary MAC. Also, the list service fetches Airtable schema before authenticating a supplied cursor. | Valid pagination can fail intermittently. Invalid cursors can consume upstream quota once a caller exposes this list operation. | Parse the fixed-length MAC. Verify it and the VA, revision, entity and filters immediately after loading the authorized document, before decrypting the PAT or fetching schema. Test that invalid cursors produce zero upstream requests. [Cursor decoder](../../../services/politburo/internal/community/virtualairlines/connections/cursor.go) · [List operation](../../../services/politburo/internal/community/virtualairlines/connections/service.go) |

## Shared module and storage contract

Place the primitive in `services/politburo/internal/secretstore`. Construct it once in the composition root and inject it into consumers. It owns the envelope, encryption, decryption and authenticated context encoding. Each consumer owns its storage format around the envelope, access checks and the choice of which fields are secret. This keeps database, Redis, Airtable and HTTP dependencies out of the module. [Composition root](../../../services/politburo/internal/app/app.go)

The proposed Go API takes plaintext bytes so consumers can also seal serialized records.

```go
type Context struct {
    Purpose  string
    OwnerID  string
    RecordID string
}

type Envelope struct {
    Version int    `json:"version"`
    Sealed  []byte `json:"sealed"`
}

func (c *Cipher) Encrypt(ctx Context, plaintext []byte) (Envelope, error)
func (c *Cipher) Decrypt(ctx Context, value Envelope) ([]byte, error)
```

- Treat `DATA_ENCRYPTION_KEY` as a deployment root. At startup derive two 32-byte subkeys with HKDF-SHA256 and fixed, distinct info strings, e.g. `politburo/at-rest/aes-gcm/v1` and `politburo/connections/cursor-hmac/v1`. Give the cipher only the AES subkey and the cursor code only the HMAC subkey. Remove the raw-key accessor. This preserves a single operator-managed setting and allows every replica to verify cursors.
- Use AES-256 with Go's `cipher.NewGCMWithRandomNonce` (Politburo targets Go 1.25). Its `Seal` result includes the random nonce and tag. JSON encodes the byte slice as one base64 string. Pass an empty nonce to this AEAD on seal and open. Reject unsupported versions, missing fields, invalid base64 and truncated blobs as errors. Return no plaintext on authentication failure.
- For a PAT, use the fixed purpose `airtable.pat`, the VA ID from the authorized operation and the selected connection ID. Encode envelope version, purpose, owner and record as an ordered JSON array for associated data. Require populated context fields and reconstruct them on each read. A copied blob then fails authentication in another VA, connection or secret field. Keep mutable names and revisions out of this context so ordinary edits retain readable credentials.
- This protects a Postgres dump or backup obtained without the deployment key. A running Politburo instance necessarily has access to plaintext for outbound provider requests. Ordinary REST responses and UI handlers do not return sealed ciphertext, envelopes, or decrypted secrets; a separate handler is required to return an encrypted value and should be rare ([response convention](../conventions.md)). Keep tokens, keys, envelopes, and authorization headers out of logs, metrics, and errors. Keep signed dashboard tickets on their existing `SIGNED_LINK_SECRET` and expiry lifecycle. [Ticket implementation](../../../services/politburo/internal/access/auth/ticket.go)

**Cost per credential.** The sealed value adds 28 bytes (12-byte nonce and 16-byte tag). For a 64-byte token, that is 92 bytes before base64 and 124 characters afterward, plus the small JSON envelope. The key is loaded once at startup. Encrypt only on token creation or replacement, and decrypt once per provider operation. Admin listings require zero cryptographic operations. The design adds no network call or database round trip.

## Implementation and dev cutover

1. Replace the connection-specific config field and parser with a service-level at-rest setting. Reuse `secret()` for `DATA_ENCRYPTION_KEY` and `DATA_ENCRYPTION_KEY_FILE`, including its mutual-exclusion check. Require standard base64 decoding to exactly 32 bytes when set. Provide no generated-on-startup or built-in fallback key. [Config loader](../../../services/politburo/internal/config/config.go)
2. Wire the shared cipher and cursor subkey in `internal/app/app.go`. Replace `EncryptedCredential` with the shared envelope, route both write paths and `tokenForConnection` through it, and update the credential-presence projection. Delete the feature-local cipher and move its relevant tests. Keep the existing revision-checked repository write and the disabled-feature HTTP 503. [Stored document and redaction](../../../services/politburo/internal/community/virtualairlines/connections/config.go) · [Repository](../../../services/politburo/internal/community/virtualairlines/config/repository.go)
3. Update the [service env template](../../../services/politburo/.env.example), [production env template](../../../infra/prod/env/politburo.env.example), [dashboard setup message](../../../services/politburo/internal/ui/static/js/admin/data-connections.mjs), [connections design](live-data-connections.md) and [product docs](../product/airtable-datasource.md). Keep example values empty. Update the 503 description in the [OpenAPI source](../../../openapi/politburo/paths/va/data-connections.yaml) and regenerate the committed bundle with `make generate`. Public request and response shapes can stay the same.
4. The unused feature needs no SQL change or backfill. If a developer has a disposable fixture, recreate the affected connection with its PAT and start pagination again. Once real credentials exist, changing the root makes their ciphertext unreadable and invalidates existing cursors. Future rotation must either coordinate credential re-entry or introduce key identification and re-encryption before retiring the previous key. Include all consumers and replicas when tracking the random-nonce GCM limit of 2³² encryptions per AES key.

## Infrastructure changes for implementation

Generate the value once per environment with `openssl rand -base64 32`. This produces 32 random bytes encoded as 44 base64 characters. Store it in the private Politburo env file (mode `0600`) and keep a recoverable copy separate from database backups. Reuse it across restarts, deploys and replicas.

| Deployment | Tracked change | Operator action |
| --- | --- | --- |
| Local dev | Add one line to the Politburo setup section in [infra/dev/README.md](../../../infra/dev/README.md). Air loads the service’s own env file. | Set `DATA_ENCRYPTION_KEY` in `services/politburo/.env` and restart Air. Dev Compose supplies the backing services only. |
| Production k3s | Replace the old setting with an empty `DATA_ENCRYPTION_KEY=` in the [production template](../../../infra/prod/env/politburo.env.example). Document the Secret update in the [k3s runbook](../../../infra/prod/k8s/README.md). The [deployment](../../../infra/prod/k8s/base/apps/politburo.yaml) already imports `politburo-env`. | Update the private `infra/prod/env/politburo.env` and the `politburo-env` Secret in `ie-apps`. Preserve other Secret entries, including cluster DNS overrides. Roll out the new Politburo image after the Secret update. For an already deployed image, use `kubectl -n ie-apps rollout restart deployment/politburo`. Check `kubectl -n ie-apps rollout status deployment/politburo`. |
| Legacy production Compose | The same production template feeds the [Politburo service](../../../infra/prod/docker-compose.prod.yml) through its existing `env_file`. | After updating the private env file and image, recreate Politburo from `infra/prod` with `podman compose -f docker-compose.prod.yml up -d --force-recreate politburo`. |

Verify delivery by creating a disposable connection, restarting Politburo and inspecting its Airtable schema again. This exercises key persistence without printing the key.

## Next consumer identified during review

Infinite Flight access and refresh tokens are another fit for the module. They predate the Airtable change and have a different lifecycle. The OAuth callback stores a user token session in Redis with `session.TTL`. VA grant creation and refresh write a VA token session with a 30-day TTL. The user live handler and the VA token provider read those records directly. The implementation already flags token encryption as a TODO. [OAuth callback](../../../services/politburo/internal/transport/http/api/oauth/handler.go) · [User token reader](../../../services/politburo/internal/transport/http/api/live/handler.go) · [VA token provider](../../../services/politburo/internal/livegame/valive/token_provider.go)

For a subsequent OAuth rollout, put a typed token-session store around these Redis reads and writes. Seal the serialized session before each write and open it after each read, including the reload after refresh. Give user and VA sessions distinct purposes and bind each envelope to the intended owner and cache key. Preserve each TTL, cache-miss behavior and Postgres grant metadata. Gate OAuth setup and refresh on cipher availability before accepting new credentials. Verify deployment usage before choosing forced reconnect or migration for existing sessions. The disposable-data assumption applies specifically to the new Airtable feature.

## Acceptance checks for implementation

| Check | Expected evidence |
| --- | --- |
| Primitive | Two seals of one token differ and both round-trip. Wrong key, purpose, owner or record fails. Modified version, nonce, ciphertext or tag, invalid base64 and short input fail without panic or plaintext. |
| Configuration and replicas | Direct value and file-backed value work independently. Conflicting settings, bad base64 and incorrect length fail startup without printing the key. With neither setting, the service starts and connections returns 503. Another instance with the same root opens the record and verifies its cursor. |
| Stored and public representations | Capture the raw JSON passed to the repository on create and replacement. It contains an envelope and no PAT. The admin response exposes only credential presence. A generic config request still cannot read or write the block. |
| Lifecycle and isolation | A stub Airtable provider receives the correct token after a restart and after replacement. Name-only edits retain the envelope. A rejected replacement or stale revision preserves the previous token. Swapping envelopes across connections or VAs fails before provider access. |
| Pagination | A cursor round-trips across instances. A modified cursor, different VA, changed filters or stale revision is rejected. Include a MAC containing a literal dot byte. The current decoder searches for the last dot in a raw binary MAC payload, so parse the fixed SHA-256 MAC length to avoid rejecting a valid cursor. |
| Repository checks | Run `go test ./...` from `services/politburo` after generation, and `make openapi-bundle-check` from the root after updating the spec description. Verify the dashboard setup message and both environment templates reference the shared setting. |

# Politburo `internal/` layout

Go code is grouped by **bounded context**. Shared runtime pieces stay at this directory’s root so imports stay short.

## Contexts

| Path | Purpose |
|------|---------|
| [`access/`](access/) | Identity for HTTP callers: API keys, JWT-style claims, signed-link tickets, browser sessions (Redis + cookie). Not business rules. |
| [`livegame/`](livegame/) | Infinite Flight **live** data: cache domains (`flights`, `sessions`, `liveries`), IF Live API client, cron scheduler, sync jobs. |
| [`community/`](community/) | VA membership, user registration, Discord-bot JSON flows: users, virtual airlines, membership (+ roster + Discord resolver), registration (+ route proof). |
| [`operations/`](operations/) | Platform operator tools: operator service, report storage. |

## Delivery and composition

| Path | Purpose |
|------|---------|
| [`app/`](app/) | **Only** composition root — constructs and wires dependencies. |
| [`transport/http/`](transport/http/) | Chi router, middleware, JSON handlers, SSR HTTP adapters. |
| [`ui/`](ui/) | Embedded templates and static assets for SSR. |
| [`api/generated/`](api/generated/) | oapi-codegen output (gitignored until `make generate`). |

## Shared runtime (root)

`config`, `cache`, `database`, `logging`, `metrics` — used by every context.

## Import rules

- **`livegame/*` and `community/*`** must not import `access/session` (cookies/sessions stay in HTTP + access). See [`docs/politburo/conventions.md`](../../../docs/politburo/conventions.md).
- **Cache keys** only via [`cache/keys.go`](cache/keys.go).
- **IF Live API** only through [`livegame/infiniteflight`](livegame/infiniteflight/). `community/registration` may use it for logbook proof; do not pull livegame cache packages from community code.
- **Jobs** register only in [`jobs/register.go`](jobs/register.go). Cron/TTL constants live in [`cache/`](cache/).

Full system map: [`.claude/commands/architecture.md`](../../../.claude/commands/architecture.md).

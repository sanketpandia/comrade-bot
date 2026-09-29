# OpenAPI workflow

## Source of truth

[`openapi/politburo.yaml`](../../openapi/politburo.yaml) is the authoritative **machine** API contract
(JSON under `/health/*` and `/api/v1/...`). Do not put SSR HTML routes in OpenAPI.

| Consumer | Output |
|----------|--------|
| Politburo Go server | `services/politburo/internal/api/generated/politburo/server.gen.go` |
| Swagger UI (local) | `http://localhost:8081` via `make openapi-view` |
| Discord bot | `services/comrade-bot-discord/src/generated/politburo-api.ts` |

Generated files are build artifacts (gitignored). Run from the repository root:

```sh
make generate      # Go server + IF client + bot TypeScript types
make check         # generate + go test (Politburo)
make openapi-view  # generate Go bindings and start Swagger UI
```

Implementation lives in [`cicd/Makefile`](../../cicd/Makefile).

## Editing the contract

1. Edit `openapi/politburo.yaml`.
2. Run `make generate`.
3. Update handlers in Politburo and HTTP calls in `services/comrade-bot-discord/src/services/apiService.ts` as needed.

Infinite Flight upstream spec: `openapi/infinite-flight/` → `services/politburo/internal/api/generated/infiniteflight/`.

## Bot local API URL

```sh
cd services/comrade-bot-discord
API_URL=http://localhost:8082 npm run dev
```

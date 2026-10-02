# OpenAPI workflow

## Source of truth

[`openapi/politburo/`](../../openapi/politburo/) is the authoritative **machine** API contract
(JSON under `/health/*` and `/api/v1/...`). Do not put SSR HTML routes in OpenAPI.

| Path | Contents |
|------|----------|
| `openapi/politburo/openapi.yaml` | Root: info, servers, path index, security schemes, schema index |
| `openapi/politburo/paths/<area>/*.yaml` | One path item per file |
| `openapi/politburo/components/schemas/*.yaml` | One schema per file; filename is the schema name |

[`openapi/politburo.yaml`](../../openapi/politburo.yaml) is the single-file bundle produced by
[Redocly CLI](https://redocly.com/docs/cli/commands/bundle) (`make openapi-bundle`). It is committed so
Docker builds and other consumers read one file without Node tooling. Do not edit it by hand;
CI fails if it is out of date with the sources.

| Consumer | Output |
|----------|--------|
| Politburo Go server | `services/politburo/internal/api/generated/politburo/server.gen.go` |
| Swagger UI (local) | `http://localhost:8081` via `make openapi-view` |
| Public ReDoc (GitHub Pages) | https://sanketpandia.github.io/comrade-bot/api/reference/ via `make docs-build` |
| Discord bot | `services/comrade-bot-discord/src/generated/politburo-api.ts` |

Generated files are build artifacts (gitignored). Run from the repository root:

```sh
make generate              # bundle spec, then Go server + IF client + bot TypeScript types
make openapi-bundle        # only rebuild openapi/politburo.yaml
make openapi-bundle-check  # fail if the committed bundle is stale
make check                 # generate + go test (Politburo)
make openapi-view          # bundle, generate Go bindings and start Swagger UI
```

Implementation lives in [`cicd/Makefile`](../../cicd/Makefile).

## Editing the contract

1. Edit files under `openapi/politburo/`. New paths and schemas must be listed in `openapi/politburo/openapi.yaml`.
   Reference schemas by relative file path (`$ref: "../../components/schemas/Flight.yaml"` from a path file,
   `$ref: "./Flight.yaml"` between schemas).
2. Run `make generate` and commit the regenerated `openapi/politburo.yaml`.
3. Update handlers in Politburo and HTTP calls in `services/comrade-bot-discord/src/services/apiService.ts` as needed.

Infinite Flight upstream spec: `openapi/infinite-flight/` → `services/politburo/internal/api/generated/infiniteflight/`.

## Bot local API URL

```sh
cd services/comrade-bot-discord
API_URL=http://localhost:8082 npm run dev
```

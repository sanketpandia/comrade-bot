# Containers and CI

Politburo has two container entry points:

- `Dockerfile.dev` provides the Go toolchain, pinned development tools, and an
  Air-based development target. Its `ci` target generates ignored OpenAPI Go
  files, runs all tests, and builds the application.
- `Dockerfile` is a multi-stage production build. OpenAPI tooling exists only
  in the disposable generation stage. Runtime dependencies are downloaded in a
  separate build stage, and the final non-root Alpine image contains only the
  application binary and CA certificates.

The GitHub Actions workflow [`.github/workflows/ci.yml`](../../../.github/workflows/ci.yml)
uses path filters to run only the jobs that match changed files.

On **pull requests** (GitHub-hosted `ubuntu-latest`):

- Contract changes: `make openapi-bundle-check`.
- Politburo: Docker `ci` target on [`Dockerfile.dev`](../../../services/politburo/Dockerfile.dev), plus a non-pushing build of the production [`Dockerfile`](../../../services/politburo/Dockerfile).
- Comrade-bot: `npm ci`, `npm run api:generate`, `npm test`, `npm run build`, `npm run commands:validate`, plus a non-pushing production image build.

On **push to `main`**: build and push GHCR images on GitHub-hosted runners
(only for affected services), then roll out deployments on the self-hosted prod
runner. No test jobs on `main`.

Images are published as `ghcr.io/<owner>/politburo:<git-sha>` and
`ghcr.io/<owner>/comrade-bot:<git-sha>` (with a floating `:main` tag).
Deployment should pull those tags rather than building on the production host.

## Path filter outputs (`changes` job)

| Output | Triggers on |
|--------|-------------|
| `contract` | `openapi/**`, `cicd/**`, root `Makefile` |
| `politburo` | `services/politburo/**`, `openapi/**`, `cicd/**`, `Makefile`, `.github/workflows/ci.yml` |
| `bot` | `services/comrade-bot-discord/**`, `openapi/**`, `cicd/**`, `Makefile`, `.github/workflows/ci.yml`, `infra/prod/scripts/k8s-sync-discord-commands.sh` |
| `infra` | `infra/prod/k8s/**`, `.github/workflows/ci.yml` |

## Development

```sh
docker build --file Dockerfile.dev --target development --tag politburo-dev .
docker run --rm --publish 8082:8082 --env-file .env politburo-dev
```

For live reload, bind-mount the checkout at `/app`. Do not copy `.env` into an
image; `.dockerignore` excludes it from every build context.

## Runtime secrets

Dockerfiles do not accept secret build arguments and do not set credentials in
image layers. Sensitive settings support either their ordinary environment
variable or a corresponding mounted-file variable:

| Secret | Mounted-file variable |
|---|---|
| `DATABASE_URL` | `DATABASE_URL_FILE` |
| `PG_PASSWORD` | `PG_PASSWORD_FILE` |
| `REDIS_PASSWORD` | `REDIS_PASSWORD_FILE` |
| `IF_API_KEY` | `IF_API_KEY_FILE` |

Prefer read-only files under `/run/secrets`, for example:

```yaml
services:
  politburo:
    image: registry.example/politburo:<git-sha>
    environment:
      PG_PASSWORD_FILE: /run/secrets/postgres_password
      REDIS_PASSWORD_FILE: /run/secrets/redis_password
      IF_API_KEY_FILE: /run/secrets/infinite_flight_api_key
    secrets:
      - postgres_password
      - redis_password
      - infinite_flight_api_key

secrets:
  postgres_password:
    file: /secure/outside/repository/postgres_password
  redis_password:
    file: /secure/outside/repository/redis_password
  infinite_flight_api_key:
    file: /secure/outside/repository/infinite_flight_api_key
```

Keep host secret files outside the repository with mode `0600`. Kubernetes
should mount equivalent Secret keys as read-only files and set the same
`*_FILE` variables. For Git-managed encrypted secrets, use SOPS with age; never
commit decoded Secret manifests or pass credentials through `docker build`.

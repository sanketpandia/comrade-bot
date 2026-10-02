# Public docs site (GitHub Pages)

User-facing guides, legal pages, and a self-hosted [ReDoc](https://github.com/Redocly/redoc)
bundle for the Politburo OpenAPI contract.

**Published URL (after setup):** <https://sanketpandia.github.io/comrade-bot/>

## One-time GitHub setup

1. Repository **Settings → Pages → Build and deployment → Source:** **GitHub Actions**.
2. After the first successful run of [`.github/workflows/pages.yml`](../.github/workflows/pages.yml)
   on `main`, the site is live at the URL above.

## Local preview

From the repository root:

```sh
make docs-build   # bundle OpenAPI, pack ReDoc, build MkDocs → docs-site/site/
make docs-serve   # pack + mkdocs serve (default http://127.0.0.1:8000)
```

Requires Node (for `docs-site/npm ci` and OpenAPI bundle in `cicd/`) and Python 3 with pip.

## Layout

| Path | Role |
|------|------|
| `docs/` | MkDocs markdown (product guides, legal) |
| `api-renderer/` | Static HTML shell for ReDoc |
| `package.json` | Pinned `redoc` — `redoc.standalone.js` copied at build time |
| `docs/api/reference/` | **Generated** — `index.html`, `redoc.standalone.js`, `openapi.yaml` |

Developer documentation for the monorepo remains in [`../docs/`](../docs/).

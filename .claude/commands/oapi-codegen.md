# oapi-codegen — Spec-Driven REST Endpoints

Use this skill when implementing any new JSON-returning REST endpoint in politburo. HTML-emitting Vizburo/dashboard template handlers are exempt — they keep the existing `http.HandlerFunc` pattern.

---

## When to use

| Situation | Approach |
|---|---|
| New REST endpoint returning JSON | oapi-codegen (this skill) |
| Extending an existing spec-driven endpoint | Extend the existing `<domain>.yaml` spec |
| Modifying an existing Swaggo-annotated handler | Keep Swaggo for now; plan spec migration only if the endpoint is being significantly reworked |
| HTML/template rendering (Vizburo, dashboard) | Existing `http.HandlerFunc` pattern — no spec |

---

## Toolchain

**Codegen CLI** (dev-time only, not a runtime dep):
```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

**Runtime deps** (add to `go.mod` once, when introducing the first spec-driven endpoint):
```
github.com/oapi-codegen/runtime
github.com/oapi-codegen/nethttp-middleware
```

---

## File layout

```
api/openapi/
  <domain>.yaml               # OpenAPI 3.0 spec — source of truth
  <domain>.cfg.yaml           # oapi-codegen config
  <domain>/generate.go        # go:generate directive (optional stub file)

internal/api/generated/
  <domain>/
    server.gen.go             # GENERATED — never hand-edit
```

One spec file per domain (e.g. `flights.yaml`, `pireps.yaml`, `users.yaml`). Not one monolithic spec. Check if a domain spec already exists before creating a new one.

---

## Spec file (`<domain>.yaml`)

Minimum required structure:
```yaml
openapi: "3.0.0"
info:
  title: "<Domain> API"
  version: "1.0.0"
paths:
  /api/v1/<resource>:
    get:
      operationId: Get<Resource>
      summary: "Brief description"
      tags: [<domain>]
      security:
        - ApiKeyAuth: []
      parameters: [...]
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/<Resource>Response"
        "400":
          $ref: "#/components/responses/ErrorResponse"
        "401":
          $ref: "#/components/responses/UnauthorizedResponse"
        "500":
          $ref: "#/components/responses/ErrorResponse"
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
  schemas:
    <Resource>Response:
      type: object
      required: [status, message]
      properties:
        status: { type: string }
        message: { type: string }
        data:
          $ref: "#/components/schemas/<Resource>"
  responses:
    ErrorResponse:
      description: Error
      content:
        application/json:
          schema:
            type: object
            properties:
              status: { type: string }
              message: { type: string }
```

The spec is the documentation — write complete `summary` and `description` fields. No Swaggo annotations needed on spec-driven handlers.

---

## Codegen config (`<domain>.cfg.yaml`)

```yaml
package: <domain>gen
generate:
  strict-server: true
  models: true
  chi-server: true
output: ../../internal/api/generated/<domain>/server.gen.go
output-options:
  skip-prune: false
```

Always use `strict-server: true` — it generates a `StrictServerInterface` with typed request/response structs, catching schema mismatches at compile time.

---

## Regenerating

Add to `politburo/Makefile`:
```makefile
generate-api:
	cd api/openapi && oapi-codegen -config <domain>.cfg.yaml <domain>.yaml
```

Or use a `go:generate` stub:
```go
// api/openapi/<domain>/generate.go
package <domain>

//go:generate oapi-codegen -config ../<domain>.cfg.yaml ../<domain>.yaml
```

Run after any spec change. The generated file must be committed alongside the spec change.

---

## Implementation pattern

The handler struct in `internal/api/<domain>_handlers.go` satisfies the generated `StrictServerInterface`:

```go
package api

import (
    "context"
    gen "infinite-experiment/politburo/internal/api/generated/<domain>"
    "infinite-experiment/politburo/internal/common"
)

type FlightHandlers struct {
    svc  FlightService
    repo FlightRepository
}

// Compile-time check
var _ gen.StrictServerInterface = (*FlightHandlers)(nil)

func (h *FlightHandlers) GetVALiveFlights(
    ctx context.Context,
    req gen.GetVALiveFlightsRequestObject,
) (gen.GetVALiveFlightsResponseObject, error) {
    initTime := time.Now()

    flights, err := h.svc.GetLive(ctx, req.Params.ServerId)
    if err != nil {
        // Still use common.RespondError internally — then wrap in generated response type
        return gen.GetVALiveFlights500JSONResponse{
            Status:  "error",
            Message: err.Error(),
        }, nil
    }

    return gen.GetVALiveFlights200JSONResponse{
        Status:  "ok",
        Message: "live flights fetched",
        Data:    flights,
    }, nil
}
```

Use `common.RespondSuccess` / `common.RespondError` for non-spec handlers. For spec-driven handlers, return the generated typed response objects directly — the strict handler wrapper handles writing the response.

---

## Mounting in Chi

In `internal/routes/` (or `api_routes.go`), inside the appropriate middleware group:

```go
import (
    gen "infinite-experiment/politburo/internal/api/generated/<domain>"
    oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
)

// Get the generated swagger spec for validation
swagger, err := gen.GetSwagger()
if err != nil {
    logging.Fatal("failed to load swagger spec", "error", err)
}
swagger.Servers = nil  // disable host validation — container IP varies

// Create strict handler (wraps implementation, handles req/resp marshaling)
strictHandler := gen.NewStrictHandler(handlers, nil)

r.Group(func(r chi.Router) {
    r.Use(oapimiddleware.OapiRequestValidatorWithOptions(swagger, &oapimiddleware.Options{
        ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
            common.RespondError(w, time.Now(), nil, message, statusCode)
        },
    }))
    r.Use(middleware.AuthMiddleware)  // existing auth chain
    gen.HandlerFromMux(strictHandler, r)
})
```

The request validator middleware automatically rejects requests that don't match the spec schema, with a 400 response — no manual validation needed in handlers.

---

## Coexistence with Swaggo

- Existing handlers keep `// @Summary` annotations and are not migrated proactively.
- New endpoints: no Swaggo annotations — the spec is the docs.
- If an existing endpoint is being significantly extended, plan migration to a spec as part of that work.
- Both the Swaggo-generated `/swagger/` UI and the OpenAPI specs can coexist; they document different endpoint sets.

---

## Checklist for a new spec-driven endpoint

- [ ] Spec exists at `api/openapi/<domain>.yaml` (new or extended)
- [ ] All response schemas defined (success + 4xx + 500)
- [ ] `operationId` set (used as function name in generated code)
- [ ] Codegen config at `api/openapi/<domain>.cfg.yaml`
- [ ] `make generate-api` runs cleanly, `server.gen.go` committed
- [ ] Handler struct implements `StrictServerInterface` (compile-time check via `var _ = ...`)
- [ ] Mounted in Chi under correct auth middleware group
- [ ] Request validation middleware wired with custom error handler
- [ ] Two runtime deps in `go.mod` if this is the first spec-driven endpoint

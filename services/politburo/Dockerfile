# syntax=docker/dockerfile:1
# Build from repository root: docker build -f services/politburo/Dockerfile .

FROM golang:1.25-alpine3.22 AS generate

RUN apk add --no-cache make
WORKDIR /repo

COPY cicd/tools/go.mod cicd/tools/go.sum ./cicd/tools/
RUN cd cicd/tools && go mod download
COPY Makefile cicd/Makefile cicd/codegen ./cicd/
COPY openapi ./openapi
COPY services/politburo/go.mod services/politburo/go.sum ./services/politburo/
RUN make -C cicd generate-politburo generate-infinite-flight

FROM golang:1.25-alpine3.22 AS build

WORKDIR /repo/services/politburo
COPY services/politburo/go.mod services/politburo/go.sum ./
RUN go mod download

COPY services/politburo/cmd ./cmd
COPY services/politburo/internal ./internal
COPY --from=generate /repo/services/politburo/internal/api/generated ./internal/api/generated

RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/politburo ./cmd/politburo

FROM alpine:3.22 AS production

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 65532 politburo \
    && adduser -S -D -H -u 65532 -G politburo politburo

COPY --from=build --chown=65532:65532 /out/politburo /usr/local/bin/politburo

USER 65532:65532
EXPOSE 8082
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --spider --quiet http://127.0.0.1:${PORT:-8082}/health/live || exit 1
ENTRYPOINT ["/usr/local/bin/politburo"]

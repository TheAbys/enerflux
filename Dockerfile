# syntax=docker/dockerfile:1.7

# Frontend assets are platform-independent; build them on the native builder platform.
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-builder

WORKDIR /src

COPY . .

WORKDIR /src/frontend

RUN npm ci
RUN npm run build


FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/api ./cmd/api

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/agent ./cmd/agent


FROM gcr.io/distroless/static-debian12:nonroot AS api

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY --from=builder /src/internal/web/templates ./internal/web/templates
COPY --from=frontend-builder /src/internal/web/static/build ./internal/web/static/build

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/api"]


FROM gcr.io/distroless/static-debian12:nonroot AS agent

WORKDIR /app

COPY --from=builder /out/agent /app/agent

USER nonroot:nonroot

ENTRYPOINT ["/app/agent"]

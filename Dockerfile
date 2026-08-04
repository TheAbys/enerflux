# syntax=docker/dockerfile:1.7

FROM golang:1.26.5-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /out/api ./cmd/api

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /out/agent ./cmd/agent


FROM gcr.io/distroless/static-debian12:nonroot AS api

WORKDIR /app

COPY --from=builder /out/api /app/api

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/api"]


FROM gcr.io/distroless/static-debian12:nonroot AS agent

WORKDIR /app

COPY --from=builder /out/agent /app/agent

USER nonroot:nonroot

ENTRYPOINT ["/app/agent"]
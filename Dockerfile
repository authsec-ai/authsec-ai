# ---------- Builder Stage ----------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git openssh ca-certificates && update-ca-certificates

ENV APPHOME=/app
WORKDIR $APPHOME

ENV GIT_TERMINAL_PROMPT=0
ENV GOPRIVATE=github.com/authsec-ai/*
ENV GONOSUMDB=github.com/authsec-ai/*
ENV GONOPROXY=github.com/authsec-ai/*

# ✅ Copy ONLY dependency files first — this layer caches until go.mod/go.sum changes
COPY go.mod go.sum ./

# ✅ Configure git auth and download deps in one step — prevents stale token in cached layer
RUN --mount=type=secret,id=github_token \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    git config --global url."https://x-access-token:$(cat /run/secrets/github_token)@github.com/".insteadOf "https://github.com/" && \
    go mod download && go mod verify

# ✅ Now copy the rest of the source (changes here won't re-trigger downloads)
COPY . ./

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /main ./cmd/main.go

# ---------- Final Stage ----------
FROM alpine:latest

RUN apk add --no-cache ca-certificates curl && update-ca-certificates

RUN addgroup -g 1000 appgroup \
 && adduser -D -u 1000 -G appgroup appuser

ENV APPHOME=/app
WORKDIR $APPHOME

COPY --from=builder /main ./
COPY --from=builder /app/migrations ./migrations

RUN chown -R 1000:1000 $APPHOME \
 && chmod +x ./main

USER 1000

EXPOSE 7468

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:7468/authsec/uflow/health || exit 1

CMD ["./main"]
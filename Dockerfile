# ---------- Builder Stage ----------
FROM golang:1.25-alpine AS builder

# Install necessary tools
RUN apk add --no-cache git openssh ca-certificates && update-ca-certificates

# Set working directory
ENV APPHOME=/app
WORKDIR $APPHOME

# ✅ Declare ARG *before* any commands that use it
ARG GITHUB_TOKEN
ARG GITHUB_USERNAME=oauth2

# ✅ Prevent interactive git prompts in CI
ENV GIT_TERMINAL_PROMPT=0

# ✅ Set Go private module envs before fetching
ENV GOPRIVATE=github.com/authsec-ai/*
ENV GONOSUMDB=github.com/authsec-ai/*
ENV GONOPROXY=github.com/authsec-ai/*

# ✅ Configure Git & netrc to use PAT for private repos
RUN git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

# ✅ Copy source after token is configured
COPY . ./

# ✅ Ensure go.sum is updated and modules are resolved
RUN go mod tidy && go mod download && go mod verify

# ✅ Build the binary for Linux AMD64
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
RUN go build -o /main ./cmd/main.go

# ---------- Final Stage ----------
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl && update-ca-certificates

# 🔐 Create non-root user with UID 1000
RUN addgroup -g 1000 appgroup \
 && adduser -D -u 1000 -G appgroup appuser

ENV APPHOME=/app
WORKDIR $APPHOME

# Copy compiled binary and required assets
COPY --from=builder /main ./
COPY --from=builder /app/migrations ./migrations

# 🔐 Fix ownership for non-root user
RUN chown -R 1000:1000 $APPHOME \
 && chmod +x ./main

# 🔐 Switch to non-root user
USER 1000

EXPOSE 7468

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:7468/authsec/uflow/health || exit 1

CMD ["./main"]

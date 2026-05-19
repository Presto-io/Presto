# Stage 1: Build Go server binary (runs on build host, cross-compiles via GOARCH)
FROM --platform=$BUILDPLATFORM golang:1.26.3-alpine AS go-builder
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /bin/presto-server ./cmd/presto-server/

# Stage 2: Build frontend (platform-independent, runs on build host)
# Set SKIP_FRONTEND_BUILD=true and pre-place frontend/build/ in context to skip npm build
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-builder
ARG SKIP_FRONTEND_BUILD=false
WORKDIR /app
COPY frontend/package*.json ./
RUN if [ "$SKIP_FRONTEND_BUILD" = "false" ]; then npm ci; fi
COPY frontend/ ./
RUN if [ "$SKIP_FRONTEND_BUILD" = "false" ]; then npm run build; fi

# Stage 3: Download typst for target arch
FROM --platform=$BUILDPLATFORM alpine:3.21 AS typst-downloader
ARG TARGETARCH
ARG TYPST_VERSION=0.14.2
# SEC-22: SHA256 checksums for typst binaries (update when TYPST_VERSION changes)
ARG TYPST_SHA256_AMD64=""
ARG TYPST_SHA256_ARM64=""
RUN apk add --no-cache curl && \
    if [ "$TARGETARCH" = "arm64" ]; then \
      TYPST_TRIPLE="aarch64-unknown-linux-musl"; \
      EXPECTED_SHA256="$TYPST_SHA256_ARM64"; \
    else \
      TYPST_TRIPLE="x86_64-unknown-linux-musl"; \
      EXPECTED_SHA256="$TYPST_SHA256_AMD64"; \
    fi && \
    curl -sSL "https://github.com/typst/typst/releases/download/v${TYPST_VERSION}/typst-${TYPST_TRIPLE}.tar.xz" \
      -o /tmp/typst.tar.xz && \
    if [ -n "$EXPECTED_SHA256" ]; then \
      echo "${EXPECTED_SHA256}  /tmp/typst.tar.xz" | sha256sum -c -; \
    else \
      echo "WARNING: No SHA256 checksum provided for typst binary. Set TYPST_SHA256_AMD64/ARM64 build args."; \
    fi && \
    tar -xJ --strip-components=1 -C /usr/local/bin/ < /tmp/typst.tar.xz && \
    rm /tmp/typst.tar.xz

# Stage 4: Download tinymist for target arch
FROM --platform=$BUILDPLATFORM alpine:3.21 AS tinymist-downloader
ARG TARGETARCH
ARG TINYMIST_VERSION=0.14.18
ARG TINYMIST_SHA256_AMD64=""
ARG TINYMIST_SHA256_ARM64=""
RUN apk add --no-cache curl tar && \
    if [ "$TARGETARCH" = "arm64" ]; then \
      TINYMIST_TRIPLE="aarch64-unknown-linux-musl"; \
      EXPECTED_SHA256="$TINYMIST_SHA256_ARM64"; \
    else \
      TINYMIST_TRIPLE="x86_64-unknown-linux-musl"; \
      EXPECTED_SHA256="$TINYMIST_SHA256_AMD64"; \
    fi && \
    curl -sSL "https://github.com/Myriad-Dreamin/tinymist/releases/download/v${TINYMIST_VERSION}/tinymist-${TINYMIST_TRIPLE}.tar.gz" \
      -o /tmp/tinymist.tar.gz && \
    if [ -n "$EXPECTED_SHA256" ]; then \
      echo "${EXPECTED_SHA256}  /tmp/tinymist.tar.gz" | sha256sum -c -; \
    else \
      echo "WARNING: No SHA256 checksum provided for tinymist binary. Set TINYMIST_SHA256_AMD64/ARM64 build args."; \
    fi && \
    mkdir -p /tmp/tinymist && \
    tar -xzf /tmp/tinymist.tar.gz -C /tmp/tinymist && \
    TINYMIST_BIN="$(find /tmp/tinymist -type f -name tinymist | head -n 1)" && \
    test -n "$TINYMIST_BIN" && \
    cp "$TINYMIST_BIN" /usr/local/bin/tinymist && \
    chmod +x /usr/local/bin/tinymist && \
    rm -rf /tmp/tinymist /tmp/tinymist.tar.gz

# Stage 5: Final image (target platform)
FROM alpine:3.21

# SEC-21: Create non-root user
RUN addgroup -S presto && adduser -S -G presto presto

COPY --from=typst-downloader /usr/local/bin/typst /usr/local/bin/typst
COPY --from=tinymist-downloader /usr/local/bin/tinymist /usr/local/bin/tinymist
COPY --from=go-builder /bin/presto-server /usr/local/bin/

# Create explicit app dirs for containers. These are intended to be mounted by users.
RUN mkdir -p /config /data/fonts /cache /logs && chown -R presto:presto /config /data /cache /logs

COPY --from=frontend-builder /app/build /srv/frontend

ENV PORT=8080
ENV HOST=0.0.0.0
ENV STATIC_DIR=/srv/frontend
ENV PRESTO_INJECT_API_KEY=true
ENV HOME=/home/presto
ENV PRESTO_CONFIG_DIR=/config
ENV PRESTO_DATA_DIR=/data
ENV PRESTO_CACHE_DIR=/cache
ENV PRESTO_LOG_DIR=/logs
EXPOSE 8080

USER presto
CMD ["presto-server"]

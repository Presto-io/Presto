# Stage 1: Build Go binaries (runs on build host, cross-compiles via GOARCH)
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS go-builder
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /bin/presto-server ./cmd/presto-server/
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /bin/presto-template-gongwen ./cmd/gongwen/
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /bin/presto-template-jiaoan-shicao ./cmd/jiaoan-shicao/

# Stage 2: Build frontend (platform-independent, runs on build host)
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

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

# Stage 4: Final image (target platform)
FROM alpine:3.21

# SEC-21: Create non-root user
RUN addgroup -S presto && adduser -S -G presto presto

COPY --from=typst-downloader /usr/local/bin/typst /usr/local/bin/typst
COPY --from=go-builder /bin/presto-server /usr/local/bin/

# Bundle templates next to server binary (server syncs them to user dir on startup)
RUN mkdir -p /usr/local/bin/templates/gongwen /usr/local/bin/templates/jiaoan-shicao
COPY --from=go-builder /bin/presto-template-gongwen /usr/local/bin/templates/gongwen/presto-template-gongwen
COPY cmd/gongwen/manifest.json /usr/local/bin/templates/gongwen/
COPY --from=go-builder /bin/presto-template-jiaoan-shicao /usr/local/bin/templates/jiaoan-shicao/presto-template-jiaoan-shicao
COPY cmd/jiaoan-shicao/manifest.json /usr/local/bin/templates/jiaoan-shicao/

# Create user data dir (will be overlaid by volume mount, server populates on startup)
RUN mkdir -p /home/presto/.presto && chown presto:presto /home/presto/.presto

COPY --from=frontend-builder /app/build /srv/frontend

ENV PORT=8080
ENV HOST=0.0.0.0
ENV STATIC_DIR=/srv/frontend
ENV HOME=/home/presto
EXPOSE 8080

USER presto
CMD ["presto-server"]

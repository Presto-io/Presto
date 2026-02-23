# Stage 1: Build Go server binary (runs on build host, cross-compiles via GOARCH)
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS go-builder
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /bin/presto-server ./cmd/presto-server/

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

# Stage 3b: Download official template binaries for target arch
FROM --platform=$BUILDPLATFORM alpine:3.21 AS template-downloader
ARG TARGETARCH
ARG TPL_VERSION=v1.0.0
RUN apk add --no-cache curl jq && \
    TEMPLATES="gongwen jiaoan-shicao" && \
    SUFFIX="linux-${TARGETARCH}" && \
    for tpl in $TEMPLATES; do \
      mkdir -p "/templates/$tpl" && \
      echo "Downloading presto-template-${tpl}-${SUFFIX}..." && \
      curl -sSL -o "/templates/$tpl/presto-template-$tpl" \
        "https://github.com/Presto-io/presto-official-templates/releases/download/${TPL_VERSION}/presto-template-${tpl}-${SUFFIX}" && \
      chmod +x "/templates/$tpl/presto-template-$tpl" && \
      "/templates/$tpl/presto-template-$tpl" --manifest > "/templates/$tpl/manifest.json"; \
    done

# Stage 4: Final image (target platform)
FROM alpine:3.21

# SEC-21: Create non-root user
RUN addgroup -S presto && adduser -S -G presto presto

COPY --from=typst-downloader /usr/local/bin/typst /usr/local/bin/typst
COPY --from=go-builder /bin/presto-server /usr/local/bin/

# Bundle templates next to server binary (server syncs them to user dir on startup)
COPY --from=template-downloader /templates/ /usr/local/bin/templates/

# Create user data dir (will be overlaid by volume mount, server populates on startup)
RUN mkdir -p /home/presto/.presto/fonts && chown -R presto:presto /home/presto/.presto

COPY --from=frontend-builder /app/build /srv/frontend

ENV PORT=8080
ENV HOST=0.0.0.0
ENV STATIC_DIR=/srv/frontend
ENV HOME=/home/presto
EXPOSE 8080

USER presto
CMD ["presto-server"]

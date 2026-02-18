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
RUN apk add --no-cache curl && \
    if [ "$TARGETARCH" = "arm64" ]; then \
      TYPST_TRIPLE="aarch64-unknown-linux-musl"; \
    else \
      TYPST_TRIPLE="x86_64-unknown-linux-musl"; \
    fi && \
    curl -sSL "https://github.com/typst/typst/releases/download/v${TYPST_VERSION}/typst-${TYPST_TRIPLE}.tar.xz" | \
    tar -xJ --strip-components=1 -C /usr/local/bin/

# Stage 4: Final image (target platform)
FROM alpine:3.21

COPY --from=typst-downloader /usr/local/bin/typst /usr/local/bin/typst
COPY --from=go-builder /bin/presto-server /usr/local/bin/

RUN mkdir -p /root/.presto/templates/gongwen /root/.presto/templates/jiaoan-shicao

COPY --from=go-builder /bin/presto-template-gongwen /root/.presto/templates/gongwen/presto-template-gongwen
COPY cmd/gongwen/manifest.json /root/.presto/templates/gongwen/

COPY --from=go-builder /bin/presto-template-jiaoan-shicao /root/.presto/templates/jiaoan-shicao/presto-template-jiaoan-shicao
COPY cmd/jiaoan-shicao/manifest.json /root/.presto/templates/jiaoan-shicao/

COPY --from=frontend-builder /app/build /srv/frontend

ENV PORT=8080
ENV STATIC_DIR=/srv/frontend
EXPOSE 8080

CMD ["presto-server"]

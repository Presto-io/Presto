# Stage 1: Build Go binaries
FROM golang:1.25-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -o /bin/presto-server ./cmd/presto-server/
RUN CGO_ENABLED=0 go build -o /bin/presto-template-gongwen ./cmd/gongwen/
RUN CGO_ENABLED=0 go build -o /bin/presto-template-jiaoan-shicao ./cmd/jiaoan-shicao/

# Stage 2: Build frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 3: Final image
FROM alpine:3.21

# Install typst
RUN apk add --no-cache curl && \
    curl -sSL https://github.com/typst/typst/releases/latest/download/typst-x86_64-unknown-linux-musl.tar.xz | \
    tar -xJ --strip-components=1 -C /usr/local/bin/ && \
    apk del curl

# Copy server
COPY --from=go-builder /bin/presto-server /usr/local/bin/

# Copy and install official templates
RUN mkdir -p /root/.presto/templates/gongwen /root/.presto/templates/jiaoan-shicao

COPY --from=go-builder /bin/presto-template-gongwen /root/.presto/templates/gongwen/presto-template-gongwen
COPY cmd/gongwen/manifest.json /root/.presto/templates/gongwen/

COPY --from=go-builder /bin/presto-template-jiaoan-shicao /root/.presto/templates/jiaoan-shicao/presto-template-jiaoan-shicao
COPY cmd/jiaoan-shicao/manifest.json /root/.presto/templates/jiaoan-shicao/

# Copy frontend
COPY --from=frontend-builder /app/build /srv/frontend

ENV PORT=8080
ENV STATIC_DIR=/srv/frontend
EXPOSE 8080

CMD ["presto-server"]

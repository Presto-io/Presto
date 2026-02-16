.PHONY: dev build desktop clean install-templates

# Build frontend
frontend:
	cd frontend && npm run build

# Build API server
server: frontend
	go build -o bin/presto-server ./cmd/presto-server/

# Build desktop app (Wails)
desktop: frontend
	cp -r frontend/build/* cmd/presto-desktop/build/
	CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags "desktop,production" -o bin/presto-desktop ./cmd/presto-desktop/

# Build template binaries
templates:
	go build -o bin/presto-template-gongwen ./cmd/gongwen/
	go build -o bin/presto-template-jiaoan-shicao ./cmd/jiaoan-shicao/

# Install templates to ~/.presto/templates/
install-templates: templates
	mkdir -p ~/.presto/templates/gongwen ~/.presto/templates/jiaoan-shicao
	cp bin/presto-template-gongwen ~/.presto/templates/gongwen/presto-template-gongwen
	cp cmd/gongwen/manifest.json ~/.presto/templates/gongwen/
	cp bin/presto-template-jiaoan-shicao ~/.presto/templates/jiaoan-shicao/presto-template-jiaoan-shicao
	cp cmd/jiaoan-shicao/manifest.json ~/.presto/templates/jiaoan-shicao/

# Build everything
build: server templates

# Run API server (dev mode)
dev:
	go run ./cmd/presto-server/

# Run desktop app
run-desktop: desktop install-templates
	./bin/presto-desktop

# Clean build artifacts
clean:
	rm -rf bin/ cmd/presto-desktop/build frontend/build

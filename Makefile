.PHONY: frontend server desktop templates install-templates build dev run-desktop clean \
       dist-macos dist-macos-arm64 dist-macos-amd64 dist-macos-universal \
       dist-dmg dist-dmg-arm64 dist-dmg-amd64 dist-dmg-universal \
       dist-windows dist-linux dist

# ─── Config ──────────────────────────────────────────────
APP_NAME     := Presto
APP_ID       := com.mrered.presto
VERSION      := 0.1.0
TYPST_VERSION:= 0.14.2
WAILS_TAGS   := desktop,production
LDFLAGS      := -s -w -X main.version=$(VERSION)
DIST         := dist
DESKTOP_SRC  := ./cmd/presto-desktop
DESKTOP_EMBED:= cmd/presto-desktop/build

# Typst download URL patterns
TYPST_BASE   := https://github.com/typst/typst/releases/download/v$(TYPST_VERSION)
TYPST_DARWIN_ARM64 := typst-aarch64-apple-darwin.tar.xz
TYPST_DARWIN_AMD64 := typst-x86_64-apple-darwin.tar.xz
TYPST_WINDOWS_AMD64:= typst-x86_64-pc-windows-msvc.zip
TYPST_LINUX_AMD64  := typst-x86_64-unknown-linux-musl.tar.xz

# ─── Development ─────────────────────────────────────────

frontend:
	cd frontend && npm run build

server: frontend
	go build -o bin/presto-server ./cmd/presto-server/

desktop: frontend
	cp -r frontend/build/* $(DESKTOP_EMBED)/
	CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -o bin/presto-desktop $(DESKTOP_SRC)/

templates:
	go build -o bin/presto-template-gongwen ./cmd/gongwen/
	go build -o bin/presto-template-jiaoan-shicao ./cmd/jiaoan-shicao/

install-templates: templates
	mkdir -p ~/.presto/templates/gongwen ~/.presto/templates/jiaoan-shicao
	cp bin/presto-template-gongwen ~/.presto/templates/gongwen/presto-template-gongwen
	cp cmd/gongwen/manifest.json ~/.presto/templates/gongwen/
	cp bin/presto-template-jiaoan-shicao ~/.presto/templates/jiaoan-shicao/presto-template-jiaoan-shicao
	cp cmd/jiaoan-shicao/manifest.json ~/.presto/templates/jiaoan-shicao/

build: server templates

dev:
	go run ./cmd/presto-server/

run-desktop: desktop install-templates
	./bin/presto-desktop

# ─── Shared ──────────────────────────────────────────────

.PHONY: _frontend-embed
_frontend-embed: frontend
	cp -r frontend/build/* $(DESKTOP_EMBED)/

# Download typst binary for a given platform
# Usage: $(MAKE) _download-typst TYPST_ARCHIVE=<name> TYPST_OUT=<path>
_download-typst:
	@mkdir -p $(dir $(TYPST_OUT))
	@if [ ! -f "$(TYPST_OUT)" ]; then \
		echo "==> Downloading typst $(TYPST_VERSION) ($(TYPST_ARCHIVE))..."; \
		TMP=$$(mktemp -d); \
		curl -sL "$(TYPST_BASE)/$(TYPST_ARCHIVE)" -o "$$TMP/archive"; \
		if echo "$(TYPST_ARCHIVE)" | grep -q '\.zip$$'; then \
			unzip -qo "$$TMP/archive" -d "$$TMP/out"; \
		else \
			mkdir -p "$$TMP/out" && tar xf "$$TMP/archive" -C "$$TMP/out"; \
		fi; \
		find "$$TMP/out" -name 'typst' -o -name 'typst.exe' | head -1 | xargs -I{} cp {} "$(TYPST_OUT)"; \
		chmod +x "$(TYPST_OUT)"; \
		rm -rf "$$TMP"; \
		echo "==> $(TYPST_OUT)"; \
	fi

# ─── macOS Distribution ─────────────────────────────────

dist-macos-arm64: _frontend-embed
	@mkdir -p $(DIST)/_bin
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" \
		-o $(DIST)/_bin/presto-darwin-arm64 $(DESKTOP_SRC)/
	@$(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_DARWIN_ARM64) TYPST_OUT=$(DIST)/_bin/typst-darwin-arm64
	@$(MAKE) _bundle-app ARCH=arm64 TYPST_BIN=$(DIST)/_bin/typst-darwin-arm64

dist-macos-amd64: _frontend-embed
	@mkdir -p $(DIST)/_bin
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" \
		-o $(DIST)/_bin/presto-darwin-amd64 $(DESKTOP_SRC)/
	@$(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_DARWIN_AMD64) TYPST_OUT=$(DIST)/_bin/typst-darwin-amd64
	@$(MAKE) _bundle-app ARCH=amd64 TYPST_BIN=$(DIST)/_bin/typst-darwin-amd64

dist-macos-universal: dist-macos-arm64 dist-macos-amd64
	@echo "==> Creating universal .app..."
	@mkdir -p "$(DIST)/$(APP_NAME)-universal.app/Contents/MacOS"
	@mkdir -p "$(DIST)/$(APP_NAME)-universal.app/Contents/Resources"
	lipo -create \
		$(DIST)/_bin/presto-darwin-arm64 \
		$(DIST)/_bin/presto-darwin-amd64 \
		-output "$(DIST)/$(APP_NAME)-universal.app/Contents/MacOS/$(APP_NAME)"
	lipo -create \
		$(DIST)/_bin/typst-darwin-arm64 \
		$(DIST)/_bin/typst-darwin-amd64 \
		-output "$(DIST)/$(APP_NAME)-universal.app/Contents/Resources/typst"
	cp packaging/macos/Info.plist "$(DIST)/$(APP_NAME)-universal.app/Contents/"
	@if [ -f packaging/macos/icon.icns ]; then \
		cp packaging/macos/icon.icns "$(DIST)/$(APP_NAME)-universal.app/Contents/Resources/"; \
	fi
	@echo "==> $(DIST)/$(APP_NAME)-universal.app"

dist-macos: dist-macos-arm64

# Internal: create .app bundle for a single arch
_bundle-app:
	@mkdir -p "$(DIST)/$(APP_NAME)-$(ARCH).app/Contents/MacOS"
	@mkdir -p "$(DIST)/$(APP_NAME)-$(ARCH).app/Contents/Resources"
	cp $(DIST)/_bin/presto-darwin-$(ARCH) \
		"$(DIST)/$(APP_NAME)-$(ARCH).app/Contents/MacOS/$(APP_NAME)"
	cp $(TYPST_BIN) "$(DIST)/$(APP_NAME)-$(ARCH).app/Contents/Resources/typst"
	cp packaging/macos/Info.plist "$(DIST)/$(APP_NAME)-$(ARCH).app/Contents/"
	@if [ -f packaging/macos/icon.icns ]; then \
		cp packaging/macos/icon.icns "$(DIST)/$(APP_NAME)-$(ARCH).app/Contents/Resources/"; \
	fi

# ─── DMG (per-arch + universal) ─────────────────────────

dist-dmg-arm64: dist-macos-arm64
	@echo "==> Creating DMG (arm64)..."
	hdiutil create -volname "$(APP_NAME)" \
		-srcfolder "$(DIST)/$(APP_NAME)-arm64.app" \
		-ov -format UDZO \
		"$(DIST)/$(APP_NAME)-$(VERSION)-macOS-arm64.dmg"
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-macOS-arm64.dmg"

dist-dmg-amd64: dist-macos-amd64
	@echo "==> Creating DMG (amd64)..."
	hdiutil create -volname "$(APP_NAME)" \
		-srcfolder "$(DIST)/$(APP_NAME)-amd64.app" \
		-ov -format UDZO \
		"$(DIST)/$(APP_NAME)-$(VERSION)-macOS-amd64.dmg"
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-macOS-amd64.dmg"

dist-dmg-universal: dist-macos-universal
	@echo "==> Creating DMG (universal)..."
	hdiutil create -volname "$(APP_NAME)" \
		-srcfolder "$(DIST)/$(APP_NAME)-universal.app" \
		-ov -format UDZO \
		"$(DIST)/$(APP_NAME)-$(VERSION)-macOS-universal.dmg"
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-macOS-universal.dmg"

dist-dmg: dist-dmg-arm64 dist-dmg-amd64 dist-dmg-universal

# ─── Windows Distribution ───────────────────────────────
# Requires: brew install mingw-w64

dist-windows-amd64: _frontend-embed
	@mkdir -p $(DIST)
	@command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1 || \
		{ echo "Error: mingw-w64 not found. Install with: brew install mingw-w64"; exit 1; }
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
		CC=x86_64-w64-mingw32-gcc \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS) -H windowsgui" \
		-o "$(DIST)/$(APP_NAME)-$(VERSION)-windows-amd64.exe" $(DESKTOP_SRC)/
	@$(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_WINDOWS_AMD64) TYPST_OUT=$(DIST)/typst.exe
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-windows-amd64.exe + typst.exe"

dist-windows: dist-windows-amd64

# ─── Linux Distribution ─────────────────────────────────
# Requires Docker (webkit2gtk headers not available on macOS)

dist-linux-amd64: frontend
	@mkdir -p $(DIST)
	@command -v docker >/dev/null 2>&1 || \
		{ echo "Error: Docker not found. Linux builds require Docker."; exit 1; }
	docker run --rm \
		-v "$(PWD)":/src \
		-w /src \
		-e GOOS=linux -e GOARCH=amd64 -e CGO_ENABLED=1 \
		golang:1.25 \
		bash -c '\
			apt-get update -qq && \
			apt-get install -y -qq libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config > /dev/null 2>&1 && \
			cp -r frontend/build/* cmd/presto-desktop/build/ && \
			go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" \
				-o dist/$(APP_NAME)-$(VERSION)-linux-amd64 $(DESKTOP_SRC)/'
	@$(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_LINUX_AMD64) TYPST_OUT=$(DIST)/typst-linux-amd64
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-linux-amd64 + typst-linux-amd64"

dist-linux: dist-linux-amd64

# ─── Build All ───────────────────────────────────────────

dist: dist-dmg dist-windows dist-linux
	@echo ""
	@echo "=== Distribution artifacts ==="
	@ls -lh $(DIST)/*.dmg $(DIST)/*.exe $(DIST)/*-linux-* 2>/dev/null || true

# ─── Clean ───────────────────────────────────────────────

clean:
	rm -rf bin/ dist/ cmd/presto-desktop/build frontend/build
	mkdir -p $(DESKTOP_EMBED)
	echo '<!doctype html>' > $(DESKTOP_EMBED)/index.html

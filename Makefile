.PHONY: frontend server desktop templates install-templates build dev run-desktop clean \
       _build-macos-arm64 _build-macos-amd64 \
       dist-macos dist-macos-arm64 dist-macos-amd64 dist-macos-universal \
       dist-dmg dist-dmg-arm64 dist-dmg-amd64 dist-dmg-universal \
       dist-windows dist-linux dist

# ─── Config ──────────────────────────────────────────────
APP_NAME     := Presto
APP_ID       := com.mrered.presto
VERSION      ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")
TYPST_VERSION:= 0.14.2
WAILS_TAGS   := desktop,production
LDFLAGS      := -s -w -X main.version=$(VERSION)
DIST         := dist
DESKTOP_SRC  := ./cmd/presto-desktop
DESKTOP_EMBED:= cmd/presto-desktop/build
MACOSX_DEPLOYMENT_TARGET := 11.0

# Typst download URL patterns
TYPST_BASE   := https://github.com/typst/typst/releases/download/v$(TYPST_VERSION)
TYPST_DARWIN_ARM64 := typst-aarch64-apple-darwin.tar.xz
TYPST_DARWIN_AMD64 := typst-x86_64-apple-darwin.tar.xz
TYPST_WINDOWS_AMD64:= typst-x86_64-pc-windows-msvc.zip
TYPST_LINUX_AMD64  := typst-x86_64-unknown-linux-musl.tar.xz

# Official templates (downloaded from GitHub Releases)
TPL_REPO     := Presto-io/presto-official-templates
TPL_VERSION  ?= v1.0.0
TPL_NAMES    := gongwen jiaoan-shicao

# ─── Development ─────────────────────────────────────────

frontend:
	cd frontend && npm run build

server: frontend
	go build -o bin/presto-server ./cmd/presto-server/

desktop: frontend
	cp -r frontend/build/* $(DESKTOP_EMBED)/
	MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" -o bin/presto-desktop $(DESKTOP_SRC)/

templates:
	@mkdir -p bin
	@for tpl in $(TPL_NAMES); do \
		echo "==> Downloading template $$tpl..."; \
		gh release download $(TPL_VERSION) --repo $(TPL_REPO) \
			-p "presto-template-$$tpl-$(shell go env GOOS)-$(shell go env GOARCH)" \
			-D bin/ --clobber; \
		mv "bin/presto-template-$$tpl-$(shell go env GOOS)-$(shell go env GOARCH)" \
			"bin/presto-template-$$tpl"; \
		chmod +x "bin/presto-template-$$tpl"; \
	done

install-templates: templates
	@for tpl in $(TPL_NAMES); do \
		mkdir -p "$$HOME/.presto/templates/$$tpl"; \
		cp "bin/presto-template-$$tpl" "$$HOME/.presto/templates/$$tpl/presto-template-$$tpl"; \
		"bin/presto-template-$$tpl" --manifest > "$$HOME/.presto/templates/$$tpl/manifest.json"; \
		echo "==> Installed $$tpl"; \
	done

build: server templates

dev:
	go run ./cmd/presto-server/

run-desktop: install-templates
	cd frontend && VITE_MOCK=1 npm run build
	cp -r frontend/build/* $(DESKTOP_EMBED)/
	rm -rf $(DESKTOP_EMBED)/mock && cp -r frontend/mock $(DESKTOP_EMBED)/mock
	MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" -o bin/presto-desktop $(DESKTOP_SRC)/
	./bin/presto-desktop

# ─── Shared ──────────────────────────────────────────────

.PHONY: _frontend-embed
ifdef SKIP_FRONTEND
_frontend-embed:
	@echo "==> Frontend pre-built (SKIP_FRONTEND=1), skipping..."
else
_frontend-embed: frontend
	cp -r frontend/build/* $(DESKTOP_EMBED)/
endif

# Download official template binaries for a given platform
# Usage: $(MAKE) _download-templates TPL_SUFFIX=<os-arch>
_download-templates:
	@mkdir -p $(DIST)/_bin
	@for tpl in $(TPL_NAMES); do \
		OUT="$(DIST)/_bin/presto-template-$$tpl-$(TPL_SUFFIX)"; \
		if [ ! -f "$$OUT" ]; then \
			echo "==> Downloading template $$tpl ($(TPL_SUFFIX))..."; \
			gh release download $(TPL_VERSION) --repo $(TPL_REPO) \
				-p "presto-template-$$tpl-$(TPL_SUFFIX)*" \
				-D $(DIST)/_bin/ --clobber; \
			chmod +x "$$OUT" 2>/dev/null || true; \
		fi; \
	done

# Extract manifest.json from a native template binary
# Usage: $(MAKE) _extract-manifests TPL_SUFFIX=<os-arch>
_extract-manifests:
	@for tpl in $(TPL_NAMES); do \
		BIN="$(DIST)/_bin/presto-template-$$tpl-$(TPL_SUFFIX)"; \
		DST="$(DIST)/_manifests/$$tpl/manifest.json"; \
		mkdir -p "$$(dirname $$DST)"; \
		"$$BIN" --manifest > "$$DST"; \
	done

# Download typst binary for a given platform
# Usage: $(MAKE) _download-typst TYPST_ARCHIVE=<name> TYPST_OUT=<path> [TYPST_SHA256=<hash>]
# SEC-22: Set TYPST_SHA256 to verify integrity of downloaded binary
_download-typst:
	@mkdir -p $(dir $(TYPST_OUT))
	@if [ ! -f "$(TYPST_OUT)" ]; then \
		echo "==> Downloading typst $(TYPST_VERSION) ($(TYPST_ARCHIVE))..."; \
		TMP=$$(mktemp -d); \
		curl -sL "$(TYPST_BASE)/$(TYPST_ARCHIVE)" -o "$$TMP/archive"; \
		if [ -n "$(TYPST_SHA256)" ]; then \
			echo "$(TYPST_SHA256)  $$TMP/archive" | shasum -a 256 -c - || \
			{ echo "ERROR: typst checksum verification failed!"; rm -rf "$$TMP"; exit 1; }; \
		else \
			echo "WARNING: No SHA256 provided. Set TYPST_SHA256 to verify integrity."; \
		fi; \
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

# Build binaries only (no .app bundle) — parallel execution
_build-macos-arm64: _frontend-embed
	@mkdir -p $(DIST)/_bin
	@( MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" \
		-o $(DIST)/_bin/presto-darwin-arm64 $(DESKTOP_SRC)/ ) & PID_GO=$$!; \
	( $(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_DARWIN_ARM64) \
		TYPST_OUT=$(DIST)/_bin/typst-darwin-arm64 ) & PID_TYPST=$$!; \
	( $(MAKE) _download-templates TPL_SUFFIX=darwin-arm64 ) & PID_TPL=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1; \
	wait $$PID_TPL || exit 1

_build-macos-amd64: _frontend-embed
	@mkdir -p $(DIST)/_bin
	@( MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" \
		-o $(DIST)/_bin/presto-darwin-amd64 $(DESKTOP_SRC)/ ) & PID_GO=$$!; \
	( $(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_DARWIN_AMD64) \
		TYPST_OUT=$(DIST)/_bin/typst-darwin-amd64 ) & PID_TYPST=$$!; \
	( $(MAKE) _download-templates TPL_SUFFIX=darwin-amd64 ) & PID_TPL=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1; \
	wait $$PID_TPL || exit 1

dist-macos-arm64: _build-macos-arm64
	@$(MAKE) _bundle-app GOARCH=arm64 TYPST_BIN=$(DIST)/_bin/typst-darwin-arm64

dist-macos-amd64: _build-macos-amd64
	@$(MAKE) _bundle-app GOARCH=amd64 TYPST_BIN=$(DIST)/_bin/typst-darwin-amd64

dist-macos-universal: _build-macos-arm64 _build-macos-amd64
	@echo "==> Extracting manifests..."
	@$(MAKE) _extract-manifests TPL_SUFFIX=darwin-arm64
	@echo "==> Creating universal .app..."
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/MacOS"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources/zh-Hans.lproj"
	@for tpl in $(TPL_NAMES); do \
		mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources/templates/$$tpl"; \
	done
	lipo -create \
		$(DIST)/_bin/presto-darwin-arm64 \
		$(DIST)/_bin/presto-darwin-amd64 \
		-output "$(DIST)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)"
	lipo -create \
		$(DIST)/_bin/typst-darwin-arm64 \
		$(DIST)/_bin/typst-darwin-amd64 \
		-output "$(DIST)/$(APP_NAME).app/Contents/Resources/typst"
	@for tpl in $(TPL_NAMES); do \
		lipo -create \
			$(DIST)/_bin/presto-template-$$tpl-darwin-arm64 \
			$(DIST)/_bin/presto-template-$$tpl-darwin-amd64 \
			-output "$(DIST)/$(APP_NAME).app/Contents/Resources/templates/$$tpl/presto-template-$$tpl"; \
		cp "$(DIST)/_manifests/$$tpl/manifest.json" \
			"$(DIST)/$(APP_NAME).app/Contents/Resources/templates/$$tpl/"; \
	done
	cp packaging/macos/Info.plist "$(DIST)/$(APP_NAME).app/Contents/"
	sed -i '' 's/0\.1\.0/$(VERSION)/g' "$(DIST)/$(APP_NAME).app/Contents/Info.plist"
	@if [ -f packaging/macos/icon.icns ]; then \
		cp packaging/macos/icon.icns "$(DIST)/$(APP_NAME).app/Contents/Resources/"; \
	fi
	codesign --force --deep -s - "$(DIST)/$(APP_NAME).app"
	@echo "==> $(DIST)/$(APP_NAME).app"

dist-macos: dist-macos-arm64

# Internal: create .app bundle for a single arch
_bundle-app:
	@$(MAKE) _extract-manifests TPL_SUFFIX=darwin-$(GOARCH)
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/MacOS"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources/zh-Hans.lproj"
	@for tpl in $(TPL_NAMES); do \
		mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources/templates/$$tpl"; \
	done
	cp $(DIST)/_bin/presto-darwin-$(GOARCH) \
		"$(DIST)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)"
	cp $(TYPST_BIN) "$(DIST)/$(APP_NAME).app/Contents/Resources/typst"
	@for tpl in $(TPL_NAMES); do \
		cp "$(DIST)/_bin/presto-template-$$tpl-darwin-$(GOARCH)" \
			"$(DIST)/$(APP_NAME).app/Contents/Resources/templates/$$tpl/presto-template-$$tpl"; \
		cp "$(DIST)/_manifests/$$tpl/manifest.json" \
			"$(DIST)/$(APP_NAME).app/Contents/Resources/templates/$$tpl/"; \
	done
	cp packaging/macos/Info.plist "$(DIST)/$(APP_NAME).app/Contents/"
	sed -i '' 's/0\.1\.0/$(VERSION)/g' "$(DIST)/$(APP_NAME).app/Contents/Info.plist"
	@if [ -f packaging/macos/icon.icns ]; then \
		cp packaging/macos/icon.icns "$(DIST)/$(APP_NAME).app/Contents/Resources/"; \
	fi
	codesign --force --deep -s - "$(DIST)/$(APP_NAME).app"

# ─── DMG (per-arch + universal) ─────────────────────────
# Requires: brew install create-dmg

DMG_BG       := packaging/macos/dmg-background.png
DMG_VOLICON  := packaging/macos/dmg-icon.icns
DMG_WIN_W    := 660
DMG_WIN_H    := 400
DMG_ICON_SIZE:= 128
DMG_APP_X    := 180
DMG_APP_Y    := 192
DMG_LNK_X   := 480
DMG_LNK_Y   := 192

define CREATE_DMG
	@echo "==> Creating DMG..."
	@command -v create-dmg >/dev/null 2>&1 || \
		{ echo "Error: create-dmg not found. Install with: brew install create-dmg"; exit 1; }
	@rm -f "$(DIST)/$(APP_NAME)-$(VERSION)-macOS-$(1).dmg"
	create-dmg \
		--volname "$(APP_NAME)" \
		--volicon "$(DMG_VOLICON)" \
		--background "$(DMG_BG)" \
		--window-pos 200 120 \
		--window-size $(DMG_WIN_W) $(DMG_WIN_H) \
		--icon-size $(DMG_ICON_SIZE) \
		--icon "$(APP_NAME).app" $(DMG_APP_X) $(DMG_APP_Y) \
		--hide-extension "$(APP_NAME).app" \
		--app-drop-link $(DMG_LNK_X) $(DMG_LNK_Y) \
		--no-internet-enable \
		"$(DIST)/$(APP_NAME)-$(VERSION)-macOS-$(1).dmg" \
		"$(DIST)/$(APP_NAME).app"
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-macOS-$(1).dmg"
endef

dist-dmg-arm64: dist-macos-arm64
	$(call CREATE_DMG,arm64)

dist-dmg-amd64: dist-macos-amd64
	$(call CREATE_DMG,amd64)

dist-dmg-universal: dist-macos-universal
	$(call CREATE_DMG,universal)

dist-dmg: dist-dmg-arm64 dist-dmg-amd64 dist-dmg-universal

# ─── Windows Distribution ───────────────────────────────
# Requires: brew install mingw-w64

dist-windows-amd64: _frontend-embed
	@mkdir -p $(DIST) $(DIST)/_bin
	@command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1 || \
		{ echo "Error: mingw-w64 not found. Install with: brew install mingw-w64"; exit 1; }
	@( GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
		CC=x86_64-w64-mingw32-gcc \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS) -H windowsgui" \
		-o "$(DIST)/$(APP_NAME)-$(VERSION)-windows.exe" $(DESKTOP_SRC)/ ) & PID_GO=$$!; \
	( $(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_WINDOWS_AMD64) \
		TYPST_OUT=$(DIST)/typst.exe ) & PID_TYPST=$$!; \
	( $(MAKE) _download-templates TPL_SUFFIX=windows-amd64.exe ) & PID_TPL=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1; \
	wait $$PID_TPL || exit 1
	@$(MAKE) _extract-manifests TPL_SUFFIX=darwin-$(shell go env GOARCH) 2>/dev/null || \
		$(MAKE) _download-templates TPL_SUFFIX=darwin-$(shell go env GOARCH) && \
		$(MAKE) _extract-manifests TPL_SUFFIX=darwin-$(shell go env GOARCH)
	@for tpl in $(TPL_NAMES); do \
		mkdir -p "$(DIST)/templates/$$tpl"; \
		cp "$(DIST)/_bin/presto-template-$$tpl-windows-amd64.exe" \
			"$(DIST)/templates/$$tpl/presto-template-$$tpl.exe"; \
		cp "$(DIST)/_manifests/$$tpl/manifest.json" "$(DIST)/templates/$$tpl/"; \
	done
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-windows.exe + typst.exe + templates/"

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
				-o dist/$(APP_NAME)-$(VERSION)-linux $(DESKTOP_SRC)/'
	@$(MAKE) _download-templates TPL_SUFFIX=linux-amd64
	@$(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_LINUX_AMD64) TYPST_OUT=$(DIST)/typst-linux
	@$(MAKE) _extract-manifests TPL_SUFFIX=darwin-$(shell go env GOARCH) 2>/dev/null || \
		$(MAKE) _download-templates TPL_SUFFIX=darwin-$(shell go env GOARCH) && \
		$(MAKE) _extract-manifests TPL_SUFFIX=darwin-$(shell go env GOARCH)
	@for tpl in $(TPL_NAMES); do \
		mkdir -p "$(DIST)/templates/$$tpl"; \
		cp "$(DIST)/_bin/presto-template-$$tpl-linux-amd64" \
			"$(DIST)/templates/$$tpl/presto-template-$$tpl"; \
		cp "$(DIST)/_manifests/$$tpl/manifest.json" "$(DIST)/templates/$$tpl/"; \
	done
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-linux + typst-linux + templates/"

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

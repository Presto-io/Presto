.PHONY: frontend server desktop build dev run-desktop check check-go check-frontend check-go-race check-desktop-compile check-local clean \
       _build-macos-arm64 _build-macos-amd64 \
       dist-macos dist-macos-arm64 dist-macos-amd64 dist-macos-universal \
       dist-dmg dist-dmg-arm64 dist-dmg-amd64 dist-dmg-universal \
       dist-windows dist-linux dist notarize inno windows-installer

# ─── Config ──────────────────────────────────────────────
APP_NAME     := Presto
APP_ID       := com.mrered.presto
VERSION      ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")
VERSION_BASE := $(shell printf '%s' "$(VERSION)" | sed 's/-.*//')
TYPST_VERSION:= 0.14.2
WAILS_TAGS   := desktop,production
LDFLAGS      := -s -w -X main.version=$(VERSION)
DIST         := dist
DESKTOP_SRC  := ./cmd/presto-desktop
DESKTOP_EMBED:= cmd/presto-desktop/build
MACOSX_DEPLOYMENT_TARGET := 11.0
INNO_COMPILER ?= ISCC.exe

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
	rm -rf $(DESKTOP_EMBED)/_app
	cp -r frontend/build/* $(DESKTOP_EMBED)/
	MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" -o bin/presto-desktop $(DESKTOP_SRC)/

build: server

check: check-go check-frontend

check-go:
	go test ./...
	go vet ./...

check-frontend:
	cd frontend && npm run check
	cd frontend && npm run build

check-go-race:
	go test ./... -race

check-desktop-compile:
	go build ./cmd/presto-desktop

check-local: check check-go-race check-desktop-compile

dev:
	go run ./cmd/presto-server/

run-desktop:
	cd frontend && VITE_MOCK=1 npm run build
	rm -rf $(DESKTOP_EMBED)/_app
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
	rm -rf $(DESKTOP_EMBED)/_app
	cp -r frontend/build/* $(DESKTOP_EMBED)/
endif

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
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1

_build-macos-amd64: _frontend-embed
	@mkdir -p $(DIST)/_bin
	@( MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" \
		-o $(DIST)/_bin/presto-darwin-amd64 $(DESKTOP_SRC)/ ) & PID_GO=$$!; \
	( $(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_DARWIN_AMD64) \
		TYPST_OUT=$(DIST)/_bin/typst-darwin-amd64 ) & PID_TYPST=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1

dist-macos-arm64: _build-macos-arm64
	@$(MAKE) _bundle-app GOARCH=arm64 TYPST_BIN=$(DIST)/_bin/typst-darwin-arm64

dist-macos-amd64: _build-macos-amd64
	@$(MAKE) _bundle-app GOARCH=amd64 TYPST_BIN=$(DIST)/_bin/typst-darwin-amd64

dist-macos-universal: _build-macos-arm64 _build-macos-amd64
	@echo "==> Creating universal .app..."
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/MacOS"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources/zh-Hans.lproj"
	lipo -create \
		$(DIST)/_bin/presto-darwin-arm64 \
		$(DIST)/_bin/presto-darwin-amd64 \
		-output "$(DIST)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)"
	lipo -create \
		$(DIST)/_bin/typst-darwin-arm64 \
		$(DIST)/_bin/typst-darwin-amd64 \
		-output "$(DIST)/$(APP_NAME).app/Contents/Resources/typst"
	cp packaging/macos/Info.plist "$(DIST)/$(APP_NAME).app/Contents/"
	cp packaging/macos/zh-Hans.lproj/InfoPlist.strings \
		"$(DIST)/$(APP_NAME).app/Contents/Resources/zh-Hans.lproj/InfoPlist.strings"
	sed -i '' 's/0\.1\.0/$(VERSION)/g' "$(DIST)/$(APP_NAME).app/Contents/Info.plist"
	@if [ -f packaging/macos/icon.icns ]; then \
		cp packaging/macos/icon.icns "$(DIST)/$(APP_NAME).app/Contents/Resources/"; \
	fi
	bash packaging/macos/codesign.sh "$(DIST)/$(APP_NAME).app"
	@echo "==> $(DIST)/$(APP_NAME).app"

dist-macos: dist-macos-arm64

# Internal: create .app bundle for a single arch
_bundle-app:
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/MacOS"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources/zh-Hans.lproj"
	cp $(DIST)/_bin/presto-darwin-$(GOARCH) \
		"$(DIST)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)"
	cp $(TYPST_BIN) "$(DIST)/$(APP_NAME).app/Contents/Resources/typst"
	cp packaging/macos/Info.plist "$(DIST)/$(APP_NAME).app/Contents/"
	cp packaging/macos/zh-Hans.lproj/InfoPlist.strings \
		"$(DIST)/$(APP_NAME).app/Contents/Resources/zh-Hans.lproj/InfoPlist.strings"
	sed -i '' 's/0\.1\.0/$(VERSION)/g' "$(DIST)/$(APP_NAME).app/Contents/Info.plist"
	@if [ -f packaging/macos/icon.icns ]; then \
		cp packaging/macos/icon.icns "$(DIST)/$(APP_NAME).app/Contents/Resources/"; \
	fi
	bash packaging/macos/codesign.sh "$(DIST)/$(APP_NAME).app"

# ─── DMG (per-arch + universal) ─────────────────────────

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
	@rm -rf "$(DIST)/_dmg_stage" && mkdir -p "$(DIST)/_dmg_stage"
	@cp -a "$(DIST)/$(APP_NAME).app" "$(DIST)/_dmg_stage/"
	bash packaging/macos/create-dmg.sh \
		--volname "$(APP_NAME)" \
		--background "$(DMG_BG)" \
		--volicon "$(DMG_VOLICON)" \
		--window-size $(DMG_WIN_W) $(DMG_WIN_H) \
		--icon-size $(DMG_ICON_SIZE) \
		--icon "$(APP_NAME).app" $(DMG_APP_X) $(DMG_APP_Y) \
		--hide-extension "$(APP_NAME).app" \
		--app-drop-link $(DMG_LNK_X) $(DMG_LNK_Y) \
		"$(DIST)/$(APP_NAME)-$(VERSION)-macOS-$(1).dmg" \
		"$(DIST)/_dmg_stage"
	@rm -rf "$(DIST)/_dmg_stage"
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-macOS-$(1).dmg"
endef

dist-dmg-arm64: dist-macos-arm64
	$(call CREATE_DMG,arm64)

dist-dmg-amd64: dist-macos-amd64
	$(call CREATE_DMG,amd64)

dist-dmg-universal: dist-macos-universal
	$(call CREATE_DMG,universal)

dist-dmg: dist-dmg-arm64 dist-dmg-amd64 dist-dmg-universal

# ─── Notarization ───────────────────────────────────────
# Usage: make notarize DMG_PATH=dist/Presto-0.1.0-macOS-arm64.dmg

notarize:
	@test -n "$(DMG_PATH)" || { echo "Error: DMG_PATH is required"; exit 1; }
	@test -f "$(DMG_PATH)" || { echo "Error: $(DMG_PATH) not found"; exit 1; }
	@echo "==> Submitting $(DMG_PATH) for notarization..."
	xcrun notarytool submit "$(DMG_PATH)" \
		--apple-id "$(APPLE_ID)" \
		--team-id "$(APPLE_TEAM_ID)" \
		--password "$(APPLE_APP_SPECIFIC_PASSWORD)" \
		--wait --timeout 10m
	@echo "==> Stapling notarization ticket..."
	xcrun stapler staple "$(DMG_PATH)"
	@echo "==> Notarization complete: $(DMG_PATH)"

# ─── Windows Distribution ───────────────────────────────
# Requires: brew install mingw-w64

dist-windows-amd64: _frontend-embed
	@mkdir -p $(DIST) $(DIST)/_bin
	@cd $(DESKTOP_SRC) && go run github.com/tc-hib/go-winres@latest make --in winres.json
	@command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1 || \
		{ echo "Error: mingw-w64 not found. Install with: brew install mingw-w64"; exit 1; }
	@( GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
		CC=x86_64-w64-mingw32-gcc \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS) -H windowsgui" \
		-o "$(DIST)/$(APP_NAME)-$(VERSION)-windows.exe" $(DESKTOP_SRC)/ ) & PID_GO=$$!; \
	( $(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_WINDOWS_AMD64) \
		TYPST_OUT=$(DIST)/typst.exe ) & PID_TYPST=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-windows.exe + typst.exe"

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
			rm -rf cmd/presto-desktop/build/_app && \
			cp -r frontend/build/* cmd/presto-desktop/build/ && \
			go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" \
				-o dist/$(APP_NAME)-$(VERSION)-linux $(DESKTOP_SRC)/'
	@$(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_LINUX_AMD64) TYPST_OUT=$(DIST)/typst-linux
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-linux + typst-linux"

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

# ─── Inno Setup Windows Installer ────────────────────────
# Note: Requires Inno Setup's ISCC compiler in PATH.

.PHONY: inno windows-installer
windows-installer: inno

inno: dist-windows-amd64
	@echo "==> Building Inno Setup installer..."
	@BINARY_PATH="$$(command -v cygpath >/dev/null 2>&1 && cygpath -w "$(PWD)/$(DIST)/$(APP_NAME)-$(VERSION)-windows.exe" || printf '%s' "$(PWD)/$(DIST)/$(APP_NAME)-$(VERSION)-windows.exe")"; \
	TYPST_PATH="$$(command -v cygpath >/dev/null 2>&1 && cygpath -w "$(PWD)/$(DIST)/typst.exe" || printf '%s' "$(PWD)/$(DIST)/typst.exe")"; \
	OUTPUT_DIR="$$(command -v cygpath >/dev/null 2>&1 && cygpath -w "$(PWD)/$(DIST)" || printf '%s' "$(PWD)/$(DIST)")"; \
	$(INNO_COMPILER) \
		"/DARG_VERSION=\"$(VERSION)\"" \
		"/DARG_FILE_VERSION=\"$(VERSION_BASE).0\"" \
		"/DARG_ARCH=\"amd64\"" \
		"/DARG_BINARY=\"$$BINARY_PATH\"" \
		"/DARG_TYPST_BINARY=\"$$TYPST_PATH\"" \
		"/DARG_OUTPUT_DIR=\"$$OUTPUT_DIR\"" \
		"/DARG_OUTPUT_BASENAME=\"$(APP_NAME)-$(VERSION)-windows-amd64-installer\"" \
		build/windows/installer/presto.iss
	@echo "==> Installer created: $(DIST)/$(APP_NAME)-$(VERSION)-windows-amd64-installer.exe"

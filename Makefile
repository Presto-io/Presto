.PHONY: frontend server desktop build dev run-desktop check check-go check-frontend check-go-race check-desktop-compile check-local clean \
       _build-macos-arm64 _build-macos-amd64 \
       _build-macos-portable-arm64 _build-macos-portable-amd64 _bundle-app-portable _frontend-embed-portable \
       dist-macos dist-macos-arm64 dist-macos-amd64 dist-macos-universal \
       dist-dmg dist-dmg-arm64 dist-dmg-amd64 dist-dmg-universal \
       dist-dmg-portable-arm64 dist-dmg-portable-amd64 dist-windows-portable-amd64 dist-linux-portable-amd64 dist-portable \
       dist-windows dist-linux dist notarize inno windows-installer _inno-language _download-typst _download-tinymist _download-vc-redist

# ─── Config ──────────────────────────────────────────────
APP_NAME     := Presto
APP_ID       := com.mrered.presto
GIT_VERSION  := $(shell git describe --tags --abbrev=0)
VERSION      ?= $(if $(GIT_VERSION),$(GIT_VERSION),0.0.0-dev)
VERSION      := $(patsubst v%,%,$(strip $(VERSION)))
VERSION_BASE := $(firstword $(subst -, ,$(VERSION)))
TYPST_VERSION:= 0.14.2
TINYMIST_VERSION := 0.14.18
WAILS_TAGS   := desktop,production
CHANNEL      ?= slim
LDFLAGS      := -s -w -X main.version=$(VERSION)
PORTABLE_LDFLAGS := $(LDFLAGS) -X main.releaseChannel=portable
PORTABLE_FRONTEND_ENV := VITE_PRESTO_CHANNEL=portable
PORTABLE_LINUX_BUILD ?= docker
# Release matrix contract: default Presto-$(VERSION)-macOS-$(1).dmg remains unchanged.
# Portable additions: Presto-$(VERSION)-portable-macOS-$(1).dmg,
# Presto-$(VERSION)-portable-windows-amd64.exe target with ZIP fallback until
# single-file embedding lands, and Presto-$(VERSION)-portable-linux-amd64.AppImage.
DIST         := dist
DESKTOP_SRC  := ./cmd/presto-desktop
DESKTOP_EMBED:= cmd/presto-desktop/build
MACOSX_DEPLOYMENT_TARGET := 11.0
INNO_COMPILER ?= ISCC.exe
INNO_LANG_DIR := build/windows/installer/languages
INNO_ZH_FILE := $(INNO_LANG_DIR)/ChineseSimplified.isl
INNO_ZH_URL := https://raw.githubusercontent.com/jrsoftware/issrc/main/Files/Languages/Unofficial/ChineseSimplified.isl
INNO_ZH_SHA256 := 7d544b9bb1d142cfa11f2e5d3cc8abe2e55f8e066c5124e3772675aa236e1278
CACHE_DIR    ?= .cache
# Typst download URL patterns
TYPST_BASE   ?= https://github.com/typst/typst/releases/download/v$(TYPST_VERSION)
TYPST_CACHE_DIR := $(CACHE_DIR)/typst/$(TYPST_VERSION)
TYPST_DARWIN_ARM64 := typst-aarch64-apple-darwin.tar.xz
TYPST_DARWIN_AMD64 := typst-x86_64-apple-darwin.tar.xz
TYPST_WINDOWS_AMD64:= typst-x86_64-pc-windows-msvc.zip
TYPST_LINUX_AMD64  := typst-x86_64-unknown-linux-musl.tar.xz
TYPST_DARWIN_ARM64_SHA256 := 470aa49a2298d20b65c119a10e4ff8808550453e0cb4d85625b89caf0cedf048
TYPST_DARWIN_AMD64_SHA256 := 4e91d8e1e33ab164f949c5762e01ee3faa585c8615a2a6bd5e3677fa8506b249
TYPST_WINDOWS_AMD64_SHA256 := 51353994ac83218c3497052e89b2c432c53b9d4439cdc1b361e2ea4798ebfc13
TYPST_LINUX_AMD64_SHA256 := a6044cbad2a954deb921167e257e120ac0a16b20339ec01121194ff9d394996d
TINYMIST_BASE := https://github.com/Myriad-Dreamin/tinymist/releases/download/v$(TINYMIST_VERSION)
TINYMIST_CACHE_DIR := $(CACHE_DIR)/tinymist/$(TINYMIST_VERSION)
TINYMIST_DARWIN_ARM64 := tinymist-aarch64-apple-darwin.tar.gz
TINYMIST_DARWIN_AMD64 := tinymist-x86_64-apple-darwin.tar.gz
TINYMIST_WINDOWS_AMD64 := tinymist-x86_64-pc-windows-msvc.zip
TINYMIST_WINDOWS_ARM64 := tinymist-aarch64-pc-windows-msvc.zip
TINYMIST_LINUX_AMD64 := tinymist-x86_64-unknown-linux-musl.tar.gz
TINYMIST_LINUX_ARM64 := tinymist-aarch64-unknown-linux-musl.tar.gz
TINYMIST_DARWIN_ARM64_SHA256 := 98d2f47e7973ff75c40e3716185385c8e7357a6a6f821771041565f222aac940
TINYMIST_DARWIN_AMD64_SHA256 := ea0ed898ea6d0fec5ccf14468d5093949585189ee4cb440d6bbb250ea57206f4
TINYMIST_WINDOWS_AMD64_SHA256 := 2c6433ce8fc5126252d5c78db3e2886d089cec7ccc5ed32f12c7b1847b534a34
TINYMIST_WINDOWS_ARM64_SHA256 := 4a302734a2a6d6c4911404ca0c62f97ace706ebc62666b18090bc36031baaaa4
TINYMIST_LINUX_AMD64_SHA256 := 6a7a2b525e9900f3718830f45a8337c5e35e55a54c8ccf7f84cd92209b924e9c
TINYMIST_LINUX_ARM64_SHA256 := 7d85fcb2c60eaa7ed7160dd68f67917f462810a6f42b1940d297e603ddc1d2ab
VC_REDIST_AMD64_URL := https://aka.ms/vc14/vc_redist.x64.exe
VC_REDIST_ARM64_URL := https://aka.ms/vc14/vc_redist.arm64.exe

# ─── Development ─────────────────────────────────────────

frontend:
	cd frontend && $(NPM) run build

server: frontend
	go build -o bin/presto-server ./cmd/presto-server/

ifeq ($(OS),Windows_NT)
DESKTOP_OUTPUT := bin/presto-desktop.exe
else
DESKTOP_OUTPUT := bin/presto-desktop
endif

# Select npm command compatible with this shell (use npm.cmd on Windows)
ifeq ($(OS),Windows_NT)
NPM := npm.cmd
else
NPM := npm
endif

desktop: frontend
ifeq ($(OS),Windows_NT)
	@"$(MAKE)" _download-typst TYPST_ARCHIVE=$(TYPST_WINDOWS_AMD64) TYPST_OUT=$(DIST)/typst.exe TYPST_SHA256=$(TYPST_WINDOWS_AMD64_SHA256)
	@"$(MAKE)" _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_WINDOWS_AMD64) TINYMIST_OUT=$(DIST)/tinymist.exe TINYMIST_SHA256=$(TINYMIST_WINDOWS_AMD64_SHA256)
	@powershell -NoProfile -Command "if (Test-Path '$(DESKTOP_EMBED)/_app') { Remove-Item -Recurse -Force '$(DESKTOP_EMBED)/_app' }"
	@powershell -NoProfile -Command "Copy-Item -Path 'frontend/build/*' -Destination '$(DESKTOP_EMBED)' -Recurse -Force"
	go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" -o $(DESKTOP_OUTPUT) $(DESKTOP_SRC)/
else
	rm -rf $(DESKTOP_EMBED)/_app
	cp -r frontend/build/* $(DESKTOP_EMBED)/
	MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" -o $(DESKTOP_OUTPUT) $(DESKTOP_SRC)/
endif

build: server

check: check-go check-frontend

check-go:
	go test ./...
	go vet ./...

check-frontend:
	cd frontend && $(NPM) run check
	cd frontend && $(NPM) run build

check-go-race:
	go test ./... -race

check-desktop-compile:
	go build ./cmd/presto-desktop

check-local: check check-go-race check-desktop-compile

dev:
	go run ./cmd/presto-server/

run-desktop:
ifeq ($(OS),Windows_NT)
	cd frontend && powershell -NoProfile -Command '$$env:VITE_MOCK="1"; $(NPM) run build'
	@"$(MAKE)" _download-typst TYPST_ARCHIVE=$(TYPST_WINDOWS_AMD64) TYPST_OUT=$(DIST)/typst.exe TYPST_SHA256=$(TYPST_WINDOWS_AMD64_SHA256)
	@"$(MAKE)" _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_WINDOWS_AMD64) TINYMIST_OUT=$(DIST)/tinymist.exe TINYMIST_SHA256=$(TINYMIST_WINDOWS_AMD64_SHA256)
	@powershell -NoProfile -Command "if (Test-Path '$(DESKTOP_EMBED)/_app') { Remove-Item -Recurse -Force '$(DESKTOP_EMBED)/_app' }"
	@powershell -NoProfile -Command "Copy-Item -Path 'frontend/build/*' -Destination '$(DESKTOP_EMBED)' -Recurse -Force"
	@powershell -NoProfile -Command "if (Test-Path '$(DESKTOP_EMBED)/mock') { Remove-Item -Recurse -Force '$(DESKTOP_EMBED)/mock' }"
	@powershell -NoProfile -Command "Copy-Item -Path 'frontend/mock' -Destination '$(DESKTOP_EMBED)' -Recurse -Force"
	go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" -o $(DESKTOP_OUTPUT) $(DESKTOP_SRC)/
	@powershell -NoProfile -Command "Start-Process -FilePath '.\\$(DESKTOP_OUTPUT)' -NoNewWindow"
else
	cd frontend && VITE_MOCK=1 npm run build
	rm -rf $(DESKTOP_EMBED)/_app
	cp -r frontend/build/* $(DESKTOP_EMBED)/
	rm -rf $(DESKTOP_EMBED)/mock && cp -r frontend/mock $(DESKTOP_EMBED)/mock
	MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" -o $(DESKTOP_OUTPUT) $(DESKTOP_SRC)/
	./$(DESKTOP_OUTPUT)
endif

# ─── Shared ──────────────────────────────────────────────
ifdef SKIP_FRONTEND
_frontend-embed:
	@echo "==> Frontend pre-built (SKIP_FRONTEND=1), skipping..."
else
_frontend-embed: frontend
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (Test-Path '$(DESKTOP_EMBED)/_app') { Remove-Item -Recurse -Force '$(DESKTOP_EMBED)/_app' }"
	@powershell -NoProfile -Command "Copy-Item -Path 'frontend/build/*' -Destination '$(DESKTOP_EMBED)' -Recurse -Force"
else
	rm -rf $(DESKTOP_EMBED)/_app
	cp -r frontend/build/* $(DESKTOP_EMBED)/
endif
endif

ifdef SKIP_FRONTEND
_frontend-embed-portable:
	@echo "==> Portable frontend pre-built (SKIP_FRONTEND=1), skipping..."
else
_frontend-embed-portable:
	cd frontend && $(PORTABLE_FRONTEND_ENV) $(NPM) run build
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (Test-Path '$(DESKTOP_EMBED)/_app') { Remove-Item -Recurse -Force '$(DESKTOP_EMBED)/_app' }"
	@powershell -NoProfile -Command "Copy-Item -Path 'frontend/build/*' -Destination '$(DESKTOP_EMBED)' -Recurse -Force"
else
	rm -rf $(DESKTOP_EMBED)/_app
	cp -r frontend/build/* $(DESKTOP_EMBED)/
endif
endif

# Download typst binary for a given platform
# Usage: $(MAKE) _download-typst TYPST_ARCHIVE=<name> TYPST_OUT=<path> [TYPST_SHA256=<hash>]
# SEC-22: Set TYPST_SHA256 to verify integrity of downloaded binary
ifeq ($(OS),Windows_NT)
_download-typst:
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path (Split-Path -Parent '$(TYPST_OUT)') | Out-Null"
	@powershell -NoProfile -Command '$$out = "$(TYPST_OUT)"; $$cache = "$(TYPST_CACHE_DIR)/$(TYPST_ARCHIVE)"; New-Item -ItemType Directory -Force -Path (Split-Path -Parent $$cache) | Out-Null; if (Test-Path $$out) { $$sig = -join ([System.IO.File]::ReadAllBytes($$out)[0..1] | ForEach-Object { [char]$$_ }); if ($$sig -ne "MZ") { if ("$(TYPST_ARCHIVE)" -like "*.zip" -and $$sig -eq "PK" -and -not (Test-Path $$cache)) { Move-Item -LiteralPath $$out -Destination $$cache -Force; Write-Host "==> Recovered cached typst archive $$cache" } else { Remove-Item -LiteralPath $$out -Force } } }'
	@powershell -NoProfile -Command 'if (-not (Test-Path "$(TYPST_OUT)")) { if ("$(REQUIRE_TYPST_SHA256)" -eq "1" -and "$(TYPST_SHA256)" -eq "") { throw "ERROR: TYPST_SHA256 is required" }; $$cache = "$(TYPST_CACHE_DIR)/$(TYPST_ARCHIVE)"; New-Item -ItemType Directory -Force -Path (Split-Path -Parent $$cache) | Out-Null; if (-not (Test-Path $$cache)) { Write-Host "==> Downloading typst $(TYPST_VERSION) ($(TYPST_ARCHIVE))..."; & curl.exe -fL --retry 5 --retry-delay 2 --connect-timeout 30 --max-time 600 "$(TYPST_BASE)/$(TYPST_ARCHIVE)" -o $$cache; if ($$LASTEXITCODE -ne 0) { Remove-Item -LiteralPath $$cache -Force -ErrorAction SilentlyContinue; exit $$LASTEXITCODE } } else { Write-Host "==> Using cached typst archive $$cache" }; if ("$(TYPST_SHA256)" -ne "") { $$hash = (Get-FileHash -Algorithm SHA256 $$cache).Hash.ToLowerInvariant(); if ($$hash -ne "$(TYPST_SHA256)") { Remove-Item -LiteralPath $$cache -Force -ErrorAction SilentlyContinue; throw "ERROR: typst checksum verification failed!" } }; $$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName()); New-Item -ItemType Directory -Path $$tmp | Out-Null; try { if ("$(TYPST_ARCHIVE)" -like "*.zip") { Expand-Archive -LiteralPath $$cache -DestinationPath $$tmp -Force } else { & "$$env:SystemRoot\System32\tar.exe" -xf $$cache -C $$tmp; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE } }; $$bin = Get-ChildItem -LiteralPath $$tmp -Recurse -File | Where-Object { $$_.Name -eq "typst.exe" -or $$_.Name -eq "typst" } | Sort-Object @{Expression = { if ($$_.Name -eq "typst.exe") { 0 } else { 1 } }}, @{Expression = "Length"; Descending = $$true} | Select-Object -First 1; if (-not $$bin) { throw "typst binary not found in archive" }; Copy-Item -LiteralPath $$bin.FullName -Destination "$(TYPST_OUT)" -Force; $$outSig = -join ([System.IO.File]::ReadAllBytes("$(TYPST_OUT)")[0..1] | ForEach-Object { [char]$$_ }); if ("$(TYPST_OUT)" -like "*.exe" -and $$outSig -ne "MZ") { Remove-Item -LiteralPath "$(TYPST_OUT)" -Force; throw "extracted typst is not a Windows executable" }; Write-Host "==> $(TYPST_OUT)" } finally { Remove-Item -LiteralPath $$tmp -Recurse -Force -ErrorAction SilentlyContinue } }'
else
_download-typst:
	@mkdir -p $(dir $(TYPST_OUT))
	@if [ ! -f "$(TYPST_OUT)" ]; then \
		mkdir -p "$(TYPST_CACHE_DIR)"; \
		CACHE="$(TYPST_CACHE_DIR)/$(TYPST_ARCHIVE)"; \
		if [ ! -f "$$CACHE" ]; then \
			echo "==> Downloading typst $(TYPST_VERSION) ($(TYPST_ARCHIVE))..."; \
			curl -fL "$(TYPST_BASE)/$(TYPST_ARCHIVE)" -o "$$CACHE.tmp" && mv "$$CACHE.tmp" "$$CACHE" || \
			{ rm -f "$$CACHE.tmp"; exit 1; }; \
		else \
			echo "==> Using cached typst archive $$CACHE"; \
		fi; \
			test "$(REQUIRE_TYPST_SHA256)" != "1" -o -n "$(TYPST_SHA256)" || { echo "ERROR: TYPST_SHA256 is required"; exit 1; }; \
			TMP=$$(mktemp -d); \
		if [ -n "$(TYPST_SHA256)" ]; then \
			echo "$(TYPST_SHA256)  $$CACHE" | shasum -a 256 -c - || \
			{ echo "ERROR: typst checksum verification failed!"; rm -f "$$CACHE"; rm -rf "$$TMP"; exit 1; }; \
		else \
			echo "WARNING: No SHA256 provided. Set TYPST_SHA256 to verify integrity."; \
		fi; \
		if echo "$(TYPST_ARCHIVE)" | grep -q '\.zip$$'; then \
			unzip -qo "$$CACHE" -d "$$TMP/out"; \
		else \
			mkdir -p "$$TMP/out" && tar xf "$$CACHE" -C "$$TMP/out"; \
		fi; \
		find "$$TMP/out" -name 'typst' -o -name 'typst.exe' | head -1 | xargs -I{} cp {} "$(TYPST_OUT)"; \
		chmod +x "$(TYPST_OUT)"; \
		rm -rf "$$TMP"; \
		echo "==> $(TYPST_OUT)"; \
	fi
endif

# Download tinymist runtime sidecar for a given platform.
# Usage: $(MAKE) _download-tinymist TINYMIST_ARCHIVE=<name> TINYMIST_OUT=<path> TINYMIST_SHA256=<hash>
ifeq ($(OS),Windows_NT)
_download-tinymist:
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path (Split-Path -Parent '$(TINYMIST_OUT)') | Out-Null"
	@powershell -NoProfile -Command 'if (-not (Test-Path "$(TINYMIST_OUT)")) { if ("$(TINYMIST_SHA256)" -eq "") { throw "ERROR: TINYMIST_SHA256 is required" }; $$cache = "$(TINYMIST_CACHE_DIR)/$(TINYMIST_ARCHIVE)"; New-Item -ItemType Directory -Force -Path (Split-Path -Parent $$cache) | Out-Null; if (-not (Test-Path $$cache)) { Write-Host "==> Downloading tinymist $(TINYMIST_VERSION) ($(TINYMIST_ARCHIVE))..."; & curl.exe -fL --retry 5 --retry-delay 2 --connect-timeout 30 --max-time 600 "$(TINYMIST_BASE)/$(TINYMIST_ARCHIVE)" -o $$cache; if ($$LASTEXITCODE -ne 0) { Remove-Item -LiteralPath $$cache -Force -ErrorAction SilentlyContinue; exit $$LASTEXITCODE } } else { Write-Host "==> Using cached tinymist archive $$cache" }; $$hash = (Get-FileHash -Algorithm SHA256 $$cache).Hash.ToLowerInvariant(); if ($$hash -ne "$(TINYMIST_SHA256)") { Remove-Item -LiteralPath $$cache -Force -ErrorAction SilentlyContinue; throw "ERROR: tinymist checksum verification failed!" }; $$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName()); New-Item -ItemType Directory -Path $$tmp | Out-Null; try { if ("$(TINYMIST_ARCHIVE)" -like "*.zip") { Expand-Archive -LiteralPath $$cache -DestinationPath $$tmp -Force } else { & "$$env:SystemRoot\System32\tar.exe" -xf $$cache -C $$tmp; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE } }; $$bin = Get-ChildItem -LiteralPath $$tmp -Recurse -File | Where-Object { $$_.Name -eq "tinymist.exe" -or $$_.Name -eq "tinymist" } | Sort-Object @{Expression = { if ($$_.Name -eq "tinymist.exe") { 0 } else { 1 } }}, @{Expression = "Length"; Descending = $$true} | Select-Object -First 1; if (-not $$bin) { throw "tinymist binary not found in archive" }; Copy-Item -LiteralPath $$bin.FullName -Destination "$(TINYMIST_OUT)" -Force; $$outSig = -join ([System.IO.File]::ReadAllBytes("$(TINYMIST_OUT)")[0..1] | ForEach-Object { [char]$$_ }); if ("$(TINYMIST_OUT)" -like "*.exe" -and $$outSig -ne "MZ") { Remove-Item -LiteralPath "$(TINYMIST_OUT)" -Force; throw "extracted tinymist is not a Windows executable" }; Write-Host "==> $(TINYMIST_OUT)" } finally { Remove-Item -LiteralPath $$tmp -Recurse -Force -ErrorAction SilentlyContinue } }'
else
_download-tinymist:
	@mkdir -p $(dir $(TINYMIST_OUT))
	@if [ ! -f "$(TINYMIST_OUT)" ]; then \
		test -n "$(TINYMIST_SHA256)" || { echo "ERROR: TINYMIST_SHA256 is required"; exit 1; }; \
		mkdir -p "$(TINYMIST_CACHE_DIR)"; \
		CACHE="$(TINYMIST_CACHE_DIR)/$(TINYMIST_ARCHIVE)"; \
		if [ ! -f "$$CACHE" ]; then \
			echo "==> Downloading tinymist $(TINYMIST_VERSION) ($(TINYMIST_ARCHIVE))..."; \
			curl -fL "$(TINYMIST_BASE)/$(TINYMIST_ARCHIVE)" -o "$$CACHE.tmp" && mv "$$CACHE.tmp" "$$CACHE" || \
			{ rm -f "$$CACHE.tmp"; exit 1; }; \
		else \
			echo "==> Using cached tinymist archive $$CACHE"; \
		fi; \
		TMP=$$(mktemp -d); \
		if command -v sha256sum >/dev/null 2>&1; then \
			echo "$(TINYMIST_SHA256)  $$CACHE" | sha256sum -c - || \
			{ echo "ERROR: tinymist checksum verification failed!"; rm -f "$$CACHE"; rm -rf "$$TMP"; exit 1; }; \
		else \
			echo "$(TINYMIST_SHA256)  $$CACHE" | shasum -a 256 -c - || \
			{ echo "ERROR: tinymist checksum verification failed!"; rm -f "$$CACHE"; rm -rf "$$TMP"; exit 1; }; \
		fi; \
		if echo "$(TINYMIST_ARCHIVE)" | grep -q '\.zip$$'; then \
			unzip -qo "$$CACHE" -d "$$TMP/out"; \
		else \
			mkdir -p "$$TMP/out" && tar xf "$$CACHE" -C "$$TMP/out"; \
		fi; \
		BIN=$$(find "$$TMP/out" -type f \( -name 'tinymist' -o -name 'tinymist.exe' \) | head -1); \
		test -n "$$BIN" || { echo "ERROR: tinymist binary not found in archive"; rm -rf "$$TMP"; exit 1; }; \
		cp "$$BIN" "$(TINYMIST_OUT)"; \
		chmod +x "$(TINYMIST_OUT)"; \
		rm -rf "$$TMP"; \
		echo "==> $(TINYMIST_OUT)"; \
	fi
endif

# Download Microsoft Visual C++ Redistributable for Windows Typst (MSVC build).
# Usage: $(MAKE) _download-vc-redist VC_REDIST_URL=<url> VC_REDIST_OUT=<path>
ifeq ($(OS),Windows_NT)
_download-vc-redist:
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path (Split-Path -Parent '$(VC_REDIST_OUT)') | Out-Null"
	@powershell -NoProfile -Command "if (-not (Test-Path '$(VC_REDIST_OUT)')) { Write-Host '==> Downloading Microsoft Visual C++ Redistributable...'; Invoke-WebRequest -UseBasicParsing -Uri '$(VC_REDIST_URL)' -OutFile '$(VC_REDIST_OUT).tmp'; Move-Item -LiteralPath '$(VC_REDIST_OUT).tmp' -Destination '$(VC_REDIST_OUT)' -Force; Write-Host '==> $(VC_REDIST_OUT)' }"
else
_download-vc-redist:
	@mkdir -p $(dir $(VC_REDIST_OUT))
	@if [ ! -f "$(VC_REDIST_OUT)" ]; then \
		echo "==> Downloading Microsoft Visual C++ Redistributable..."; \
		curl -fsSL "$(VC_REDIST_URL)" -o "$(VC_REDIST_OUT).tmp"; \
		mv "$(VC_REDIST_OUT).tmp" "$(VC_REDIST_OUT)"; \
		echo "==> $(VC_REDIST_OUT)"; \
	fi
endif

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
			TYPST_OUT=$(DIST)/_bin/typst-darwin-arm64 TYPST_SHA256=$(TYPST_DARWIN_ARM64_SHA256) REQUIRE_TYPST_SHA256=1 ) & PID_TYPST=$$!; \
	( $(MAKE) _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_DARWIN_ARM64) \
		TINYMIST_SHA256=$(TINYMIST_DARWIN_ARM64_SHA256) \
		TINYMIST_OUT=$(DIST)/_bin/tinymist-darwin-arm64 ) & PID_TINYMIST=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1; \
	wait $$PID_TINYMIST || exit 1

_build-macos-amd64: _frontend-embed
	@mkdir -p $(DIST)/_bin
	@( MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" \
		-o $(DIST)/_bin/presto-darwin-amd64 $(DESKTOP_SRC)/ ) & PID_GO=$$!; \
		( $(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_DARWIN_AMD64) \
			TYPST_OUT=$(DIST)/_bin/typst-darwin-amd64 TYPST_SHA256=$(TYPST_DARWIN_AMD64_SHA256) REQUIRE_TYPST_SHA256=1 ) & PID_TYPST=$$!; \
	( $(MAKE) _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_DARWIN_AMD64) \
		TINYMIST_SHA256=$(TINYMIST_DARWIN_AMD64_SHA256) \
		TINYMIST_OUT=$(DIST)/_bin/tinymist-darwin-amd64 ) & PID_TINYMIST=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1; \
	wait $$PID_TINYMIST || exit 1

dist-macos-arm64: _build-macos-arm64
	@$(MAKE) _bundle-app GOARCH=arm64 TYPST_BIN=$(DIST)/_bin/typst-darwin-arm64 TINYMIST_BIN=$(DIST)/_bin/tinymist-darwin-arm64

dist-macos-amd64: _build-macos-amd64
	@$(MAKE) _bundle-app GOARCH=amd64 TYPST_BIN=$(DIST)/_bin/typst-darwin-amd64 TINYMIST_BIN=$(DIST)/_bin/tinymist-darwin-amd64

dist-macos-universal: _build-macos-arm64 _build-macos-amd64
	@echo "==> Creating universal .app..."
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/MacOS"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources/zh-Hans.lproj"
	rm -rf "$(DIST)/$(APP_NAME).app/Contents/Resources/sidecars/tinymist"
	lipo -create \
		$(DIST)/_bin/presto-darwin-arm64 \
		$(DIST)/_bin/presto-darwin-amd64 \
		-output "$(DIST)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)"
	lipo -create \
		$(DIST)/_bin/typst-darwin-arm64 \
		$(DIST)/_bin/typst-darwin-amd64 \
		-output "$(DIST)/$(APP_NAME).app/Contents/Resources/typst"
	lipo -create \
		$(DIST)/_bin/tinymist-darwin-arm64 \
		$(DIST)/_bin/tinymist-darwin-amd64 \
		-output "$(DIST)/$(APP_NAME).app/Contents/Resources/tinymist"
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
	rm -rf "$(DIST)/$(APP_NAME).app/Contents/Resources/sidecars/tinymist"
	cp $(DIST)/_bin/presto-darwin-$(GOARCH) \
		"$(DIST)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)"
	cp $(TYPST_BIN) "$(DIST)/$(APP_NAME).app/Contents/Resources/typst"
	cp $(TINYMIST_BIN) "$(DIST)/$(APP_NAME).app/Contents/Resources/tinymist"
	chmod +x "$(DIST)/$(APP_NAME).app/Contents/Resources/tinymist"
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

_build-macos-portable-arm64: _frontend-embed-portable
	@mkdir -p $(DIST)/_bin
	@( MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(PORTABLE_LDFLAGS)" \
		-o $(DIST)/_bin/presto-portable-darwin-arm64 $(DESKTOP_SRC)/ ) & PID_GO=$$!; \
		( $(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_DARWIN_ARM64) \
			TYPST_OUT=$(DIST)/_bin/typst-darwin-arm64 TYPST_SHA256=$(TYPST_DARWIN_ARM64_SHA256) REQUIRE_TYPST_SHA256=1 ) & PID_TYPST=$$!; \
	( $(MAKE) _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_DARWIN_ARM64) \
		TINYMIST_SHA256=$(TINYMIST_DARWIN_ARM64_SHA256) \
		TINYMIST_OUT=$(DIST)/_bin/tinymist-darwin-arm64 ) & PID_TINYMIST=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1; \
	wait $$PID_TINYMIST || exit 1

_build-macos-portable-amd64: _frontend-embed-portable
	@mkdir -p $(DIST)/_bin
	@( MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
		CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(PORTABLE_LDFLAGS)" \
		-o $(DIST)/_bin/presto-portable-darwin-amd64 $(DESKTOP_SRC)/ ) & PID_GO=$$!; \
		( $(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_DARWIN_AMD64) \
			TYPST_OUT=$(DIST)/_bin/typst-darwin-amd64 TYPST_SHA256=$(TYPST_DARWIN_AMD64_SHA256) REQUIRE_TYPST_SHA256=1 ) & PID_TYPST=$$!; \
	( $(MAKE) _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_DARWIN_AMD64) \
		TINYMIST_SHA256=$(TINYMIST_DARWIN_AMD64_SHA256) \
		TINYMIST_OUT=$(DIST)/_bin/tinymist-darwin-amd64 ) & PID_TINYMIST=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1; \
	wait $$PID_TINYMIST || exit 1

_bundle-app-portable:
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/MacOS"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources"
	@mkdir -p "$(DIST)/$(APP_NAME).app/Contents/Resources/zh-Hans.lproj"
	rm -rf "$(DIST)/$(APP_NAME).app/Contents/Resources/sidecars/tinymist"
	cp $(DIST)/_bin/presto-portable-darwin-$(GOARCH) \
		"$(DIST)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)"
	cp $(TYPST_BIN) "$(DIST)/$(APP_NAME).app/Contents/Resources/typst"
	cp $(TINYMIST_BIN) "$(DIST)/$(APP_NAME).app/Contents/Resources/tinymist"
	chmod +x "$(DIST)/$(APP_NAME).app/Contents/Resources/typst" "$(DIST)/$(APP_NAME).app/Contents/Resources/tinymist"
	bash packaging/release/portable-templates.sh "$(DIST)/$(APP_NAME).app/Contents/Resources"
	cp packaging/macos/Info.plist "$(DIST)/$(APP_NAME).app/Contents/"
	cp packaging/macos/zh-Hans.lproj/InfoPlist.strings \
		"$(DIST)/$(APP_NAME).app/Contents/Resources/zh-Hans.lproj/InfoPlist.strings"
	sed -i '' 's/0\.1\.0/$(VERSION)/g' "$(DIST)/$(APP_NAME).app/Contents/Info.plist"
	@if [ -f packaging/macos/icon.icns ]; then \
		cp packaging/macos/icon.icns "$(DIST)/$(APP_NAME).app/Contents/Resources/"; \
	fi
	bash packaging/macos/codesign.sh "$(DIST)/$(APP_NAME).app"

define CREATE_PORTABLE_DMG
	@echo "==> Creating portable DMG..."
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
		"$(DIST)/$(APP_NAME)-$(VERSION)-portable-macOS-$(1).dmg" \
		"$(DIST)/_dmg_stage"
	@rm -rf "$(DIST)/_dmg_stage"
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-portable-macOS-$(1).dmg"
endef

dist-dmg-portable-arm64: _build-macos-portable-arm64
	@$(MAKE) _bundle-app-portable GOARCH=arm64 TYPST_BIN=$(DIST)/_bin/typst-darwin-arm64 TINYMIST_BIN=$(DIST)/_bin/tinymist-darwin-arm64
	$(call CREATE_PORTABLE_DMG,arm64)

dist-dmg-portable-amd64: _build-macos-portable-amd64
	@$(MAKE) _bundle-app-portable GOARCH=amd64 TYPST_BIN=$(DIST)/_bin/typst-darwin-amd64 TINYMIST_BIN=$(DIST)/_bin/tinymist-darwin-amd64
	$(call CREATE_PORTABLE_DMG,amd64)

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
# Native Windows builds do not require mingw-w64.
# Cross-compiling Windows binaries from Unix-like systems requires mingw-w64.
#   macOS: brew install mingw-w64

ifeq ($(OS),Windows_NT)
dist-windows-amd64: _frontend-embed
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(DIST)','$(DIST)\\_bin' | Out-Null"
	@powershell -NoProfile -Command "Remove-Item -Force '$(DIST)\\$(APP_NAME)-$(VERSION)-windows.exe' -ErrorAction SilentlyContinue; if (Test-Path '$(DIST)\\$(APP_NAME)-$(VERSION)-windows.exe') { throw 'Cannot replace $(DIST)\\$(APP_NAME)-$(VERSION)-windows.exe. Close the running Presto app and retry.' }"
	@"$(MAKE)" _download-typst TYPST_ARCHIVE=$(TYPST_WINDOWS_AMD64) TYPST_OUT=$(DIST)/typst.exe TYPST_SHA256=$(TYPST_WINDOWS_AMD64_SHA256) REQUIRE_TYPST_SHA256=1
	@"$(MAKE)" _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_WINDOWS_AMD64) TINYMIST_OUT=$(DIST)/tinymist.exe TINYMIST_SHA256=$(TINYMIST_WINDOWS_AMD64_SHA256)
	@"$(MAKE)" _download-vc-redist VC_REDIST_URL=$(VC_REDIST_AMD64_URL) VC_REDIST_OUT=$(DIST)/vc_redist.x64.exe
	@cd $(DESKTOP_SRC) && go run github.com/tc-hib/go-winres@latest make --in winres.json
	@powershell -NoProfile -Command '$$env:GOOS="windows"; $$env:GOARCH="amd64"; $$env:CGO_ENABLED="0"; & go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS) -H windowsgui" -o "$(DIST)\\$(APP_NAME)-$(VERSION)-windows.exe" "$(DESKTOP_SRC)/"; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE }'
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-windows.exe + typst.exe + tinymist.exe + vc_redist.x64.exe"
else
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
			TYPST_OUT=$(DIST)/typst.exe TYPST_SHA256=$(TYPST_WINDOWS_AMD64_SHA256) REQUIRE_TYPST_SHA256=1 ) & PID_TYPST=$$!; \
	( $(MAKE) _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_WINDOWS_AMD64) \
		TINYMIST_OUT=$(DIST)/tinymist.exe TINYMIST_SHA256=$(TINYMIST_WINDOWS_AMD64_SHA256) ) & PID_TINYMIST=$$!; \
	( $(MAKE) _download-vc-redist VC_REDIST_URL=$(VC_REDIST_AMD64_URL) \
		VC_REDIST_OUT=$(DIST)/vc_redist.x64.exe ) & PID_VC=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1; \
	wait $$PID_TINYMIST || exit 1; \
	wait $$PID_VC || exit 1
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-windows.exe + typst.exe + tinymist.exe + vc_redist.x64.exe"
endif

dist-windows: dist-windows-amd64

# Windows single-file portable exe embedding is not implemented yet. This target keeps
# the preferred portable-windows make target and emits an explicit ZIP fallback bundle
# with the exe, Typst, Tinymist, and official template snapshot.
ifeq ($(OS),Windows_NT)
dist-windows-portable-amd64: _frontend-embed-portable
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(DIST)','$(DIST)\\_portable-windows-amd64' | Out-Null"
	@"$(MAKE)" _download-typst TYPST_ARCHIVE=$(TYPST_WINDOWS_AMD64) TYPST_OUT=$(DIST)/typst.exe TYPST_SHA256=$(TYPST_WINDOWS_AMD64_SHA256) REQUIRE_TYPST_SHA256=1
	@"$(MAKE)" _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_WINDOWS_AMD64) TINYMIST_OUT=$(DIST)/tinymist.exe TINYMIST_SHA256=$(TINYMIST_WINDOWS_AMD64_SHA256)
	@cd $(DESKTOP_SRC) && go run github.com/tc-hib/go-winres@latest make --in winres.json
	@powershell -NoProfile -Command '$$env:GOOS="windows"; $$env:GOARCH="amd64"; $$env:CGO_ENABLED="0"; & go build -tags "$(WAILS_TAGS)" -ldflags "$(PORTABLE_LDFLAGS) -H windowsgui" -o "$(DIST)\\_portable-windows-amd64\\$(APP_NAME).exe" "$(DESKTOP_SRC)/"; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE }'
	@powershell -NoProfile -Command "Copy-Item '$(DIST)\\typst.exe','$(DIST)\\tinymist.exe' -Destination '$(DIST)\\_portable-windows-amd64' -Force"
	bash packaging/release/portable-templates.sh "$(DIST)/_portable-windows-amd64"
	@powershell -NoProfile -Command "Remove-Item -Force '$(DIST)\\$(APP_NAME)-$(VERSION)-portable-windows-amd64.zip' -ErrorAction SilentlyContinue; Compress-Archive -Path '$(DIST)\\_portable-windows-amd64\\*' -DestinationPath '$(DIST)\\$(APP_NAME)-$(VERSION)-portable-windows-amd64.zip' -Force"
	@echo "WARNING: $(APP_NAME)-$(VERSION)-portable-windows-amd64.exe single-file embedding is not implemented; emitted explicit portable ZIP fallback."
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-portable-windows-amd64.zip"
else
dist-windows-portable-amd64: _frontend-embed-portable
	@mkdir -p $(DIST) $(DIST)/_portable-windows-amd64
	@cd $(DESKTOP_SRC) && go run github.com/tc-hib/go-winres@latest make --in winres.json
	@command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1 || \
		{ echo "Error: mingw-w64 not found. Install with: brew install mingw-w64"; exit 1; }
	@( GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
		CC=x86_64-w64-mingw32-gcc \
		go build -tags "$(WAILS_TAGS)" -ldflags "$(PORTABLE_LDFLAGS) -H windowsgui" \
		-o "$(DIST)/_portable-windows-amd64/$(APP_NAME).exe" $(DESKTOP_SRC)/ ) & PID_GO=$$!; \
		( $(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_WINDOWS_AMD64) \
			TYPST_OUT=$(DIST)/_portable-windows-amd64/typst.exe TYPST_SHA256=$(TYPST_WINDOWS_AMD64_SHA256) REQUIRE_TYPST_SHA256=1 ) & PID_TYPST=$$!; \
	( $(MAKE) _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_WINDOWS_AMD64) \
		TINYMIST_OUT=$(DIST)/_portable-windows-amd64/tinymist.exe TINYMIST_SHA256=$(TINYMIST_WINDOWS_AMD64_SHA256) ) & PID_TINYMIST=$$!; \
	wait $$PID_GO || exit 1; \
	wait $$PID_TYPST || exit 1; \
	wait $$PID_TINYMIST || exit 1
	bash packaging/release/portable-templates.sh "$(DIST)/_portable-windows-amd64"
	@rm -f "$(DIST)/$(APP_NAME)-$(VERSION)-portable-windows-amd64.zip"
	@( cd "$(DIST)/_portable-windows-amd64" && zip -qr "../$(APP_NAME)-$(VERSION)-portable-windows-amd64.zip" . )
	@echo "WARNING: $(APP_NAME)-$(VERSION)-portable-windows-amd64.exe single-file embedding is not implemented; emitted explicit portable ZIP fallback."
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-portable-windows-amd64.zip"
endif

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
		golang:1.26.3 \
		bash -c '\
			apt-get update -qq && \
			apt-get install -y -qq libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config > /dev/null 2>&1 && \
			rm -rf cmd/presto-desktop/build/_app && \
			cp -r frontend/build/* cmd/presto-desktop/build/ && \
			go build -tags "$(WAILS_TAGS)" -ldflags "$(LDFLAGS)" \
				-o dist/$(APP_NAME)-$(VERSION)-linux $(DESKTOP_SRC)/'
	@$(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_LINUX_AMD64) TYPST_OUT=$(DIST)/typst TYPST_SHA256=$(TYPST_LINUX_AMD64_SHA256) REQUIRE_TYPST_SHA256=1
	@$(MAKE) _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_LINUX_AMD64) TINYMIST_OUT=$(DIST)/tinymist TINYMIST_SHA256=$(TINYMIST_LINUX_AMD64_SHA256)
	@echo "==> $(DIST)/$(APP_NAME)-$(VERSION)-linux + typst + tinymist"

dist-linux: dist-linux-amd64

dist-linux-portable-amd64: _frontend-embed-portable
	@mkdir -p $(DIST)
	@rm -rf "$(DIST)/_appimage"
	@mkdir -p "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/bin" "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/share/presto"
	@if [ "$(PORTABLE_LINUX_BUILD)" = "native" ]; then \
		rm -rf cmd/presto-desktop/build/_app && \
		cp -r frontend/build/* cmd/presto-desktop/build/ && \
		GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -tags "$(WAILS_TAGS)" -ldflags "$(PORTABLE_LDFLAGS)" \
			-o dist/_appimage/$(APP_NAME).AppDir/usr/bin/$(APP_NAME) $(DESKTOP_SRC)/; \
	else \
		command -v docker >/dev/null 2>&1 || \
			{ echo "Error: Docker not found. Linux portable builds require Docker."; exit 1; }; \
		docker run --rm \
			-v "$(PWD)":/src \
			-w /src \
			-e GOOS=linux -e GOARCH=amd64 -e CGO_ENABLED=1 \
			golang:1.26.3 \
			bash -c '\
				apt-get update -qq && \
				apt-get install -y -qq libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config > /dev/null 2>&1 && \
				rm -rf cmd/presto-desktop/build/_app && \
				cp -r frontend/build/* cmd/presto-desktop/build/ && \
				go build -tags "$(WAILS_TAGS)" -ldflags "$(PORTABLE_LDFLAGS)" \
					-o dist/_appimage/$(APP_NAME).AppDir/usr/bin/$(APP_NAME) $(DESKTOP_SRC)/'; \
	fi
	@$(MAKE) _download-typst TYPST_ARCHIVE=$(TYPST_LINUX_AMD64) TYPST_OUT=$(DIST)/_appimage/$(APP_NAME).AppDir/usr/bin/typst TYPST_SHA256=$(TYPST_LINUX_AMD64_SHA256) REQUIRE_TYPST_SHA256=1
	@$(MAKE) _download-tinymist TINYMIST_ARCHIVE=$(TINYMIST_LINUX_AMD64) TINYMIST_OUT=$(DIST)/_appimage/$(APP_NAME).AppDir/usr/bin/tinymist TINYMIST_SHA256=$(TINYMIST_LINUX_AMD64_SHA256)
	bash packaging/release/portable-templates.sh "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/share/presto"
	cp -R "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/share/presto/templates" "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/bin/templates"
	cp packaging/linux/presto.desktop "$(DIST)/_appimage/$(APP_NAME).AppDir/presto.desktop"
	@mkdir -p "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/share/applications" "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/share/icons/hicolor/512x512/apps"
	cp packaging/linux/presto.desktop "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/share/applications/presto.desktop"
	@if [ -f frontend/static/icon-512x512.png ]; then \
		cp frontend/static/icon-512x512.png "$(DIST)/_appimage/$(APP_NAME).AppDir/presto.png"; \
		cp frontend/static/icon-512x512.png "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/share/icons/hicolor/512x512/apps/presto.png"; \
	fi
	printf '#!/bin/sh\nHERE=$$(dirname "$$(readlink -f "$$0")")\nexec "$$HERE/usr/bin/$(APP_NAME)" "$$@"\n' > "$(DIST)/_appimage/$(APP_NAME).AppDir/AppRun"
	chmod +x "$(DIST)/_appimage/$(APP_NAME).AppDir/AppRun" "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/bin/$(APP_NAME)" "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/bin/typst" "$(DIST)/_appimage/$(APP_NAME).AppDir/usr/bin/tinymist"
	bash packaging/linux/appimage.sh "$(DIST)/_appimage/$(APP_NAME).AppDir" "$(DIST)/$(APP_NAME)-$(VERSION)-portable-linux-amd64.AppImage"

# ─── Build All ───────────────────────────────────────────

dist: dist-dmg dist-windows dist-linux
	@echo ""
	@echo "=== Distribution artifacts ==="
	@ls -lh $(DIST)/*.dmg $(DIST)/*.exe $(DIST)/*-linux-* 2>/dev/null || true

dist-portable: dist-dmg-portable-arm64 dist-dmg-portable-amd64 dist-windows-portable-amd64 dist-linux-portable-amd64
	@echo ""
	@echo "=== Portable distribution artifacts ==="
	@ls -lh $(DIST)/*portable* 2>/dev/null || true

# ─── Clean ───────────────────────────────────────────────

clean:
	rm -rf bin/ dist/ cmd/presto-desktop/build frontend/build
	mkdir -p $(DESKTOP_EMBED)
	echo '<!doctype html>' > $(DESKTOP_EMBED)/index.html

# ─── Inno Setup Windows Installer ────────────────────────
# Note: Requires Inno Setup's ISCC compiler in PATH.

.PHONY: inno windows-installer
windows-installer: inno

.PHONY: _inno-language
ifeq ($(OS),Windows_NT)
_inno-language:
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(INNO_LANG_DIR)' | Out-Null"
	@powershell -NoProfile -Command 'if (-not (Test-Path "$(INNO_ZH_FILE)")) { Write-Host "==> Downloading Inno Setup Simplified Chinese language file..."; Invoke-WebRequest -UseBasicParsing -Uri "$(INNO_ZH_URL)" -OutFile "$(INNO_ZH_FILE).tmp"; $$hash = (Get-FileHash -Algorithm SHA256 "$(INNO_ZH_FILE).tmp").Hash.ToLowerInvariant(); if ($$hash -ne "$(INNO_ZH_SHA256)") { Remove-Item -Force "$(INNO_ZH_FILE).tmp" -ErrorAction SilentlyContinue; throw "ERROR: checksum mismatch for $(INNO_ZH_FILE)" }; Move-Item -LiteralPath "$(INNO_ZH_FILE).tmp" -Destination "$(INNO_ZH_FILE)" -Force }'
	@powershell -NoProfile -Command '$$hash = (Get-FileHash -Algorithm SHA256 "$(INNO_ZH_FILE)").Hash.ToLowerInvariant(); if ($$hash -ne "$(INNO_ZH_SHA256)") { throw "ERROR: checksum mismatch for $(INNO_ZH_FILE)" }'
else
_inno-language:
	@mkdir -p "$(INNO_LANG_DIR)"
	@if [ ! -f "$(INNO_ZH_FILE)" ]; then \
		echo "==> Downloading Inno Setup Simplified Chinese language file..."; \
		curl -fsSL "$(INNO_ZH_URL)" -o "$(INNO_ZH_FILE).tmp"; \
		if command -v sha256sum >/dev/null 2>&1; then \
			echo "$(INNO_ZH_SHA256)  $(INNO_ZH_FILE).tmp" | sha256sum -c -; \
		else \
			echo "$(INNO_ZH_SHA256)  $(INNO_ZH_FILE).tmp" | shasum -a 256 -c -; \
		fi; \
		mv "$(INNO_ZH_FILE).tmp" "$(INNO_ZH_FILE)"; \
	fi
	@if command -v sha256sum >/dev/null 2>&1; then \
		echo "$(INNO_ZH_SHA256)  $(INNO_ZH_FILE)" | sha256sum -c -; \
	else \
		echo "$(INNO_ZH_SHA256)  $(INNO_ZH_FILE)" | shasum -a 256 -c -; \
	fi
endif

ifeq ($(OS),Windows_NT)
inno: dist-windows-amd64 _inno-language
	@echo "==> Building Inno Setup installer..."
	@powershell -NoProfile -Command '$$compiler = (Get-Command "$(INNO_COMPILER)" -ErrorAction SilentlyContinue).Source; if (-not $$compiler) { $$candidates = @("C:\Program Files (x86)\Inno Setup 6\ISCC.exe", "C:\Program Files\Inno Setup 6\ISCC.exe", "$$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe"); $$compiler = $$candidates | Where-Object { Test-Path $$_ } | Select-Object -First 1 }; if (-not $$compiler) { throw "Inno Setup compiler not found. Install Inno Setup 6 or run make with INNO_COMPILER=C:\path\to\ISCC.exe" }; $$binaryPath = (Resolve-Path "$(DIST)\\$(APP_NAME)-$(VERSION)-windows.exe").Path; $$typstPath = (Resolve-Path "$(DIST)\\typst.exe").Path; $$tinymistPath = (Resolve-Path "$(DIST)\\tinymist.exe").Path; $$vcRedistPath = (Resolve-Path "$(DIST)\\vc_redist.x64.exe").Path; $$outputDir = (Resolve-Path "$(DIST)").Path; & $$compiler "/DARG_VERSION=`"$(VERSION)`"" "/DARG_FILE_VERSION=`"$(VERSION_BASE).0`"" "/DARG_ARCH=`"amd64`"" "/DARG_BINARY=`"$$binaryPath`"" "/DARG_TYPST_BINARY=`"$$typstPath`"" "/DARG_TINYMIST_BINARY=`"$$tinymistPath`"" "/DARG_VC_REDIST=`"$$vcRedistPath`"" "/DARG_OUTPUT_DIR=`"$$outputDir`"" "/DARG_OUTPUT_BASENAME=`"$(APP_NAME)-$(VERSION)-windows-amd64-installer`"" "build/windows/installer/presto.iss"; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE }; if (-not (Test-Path "$(DIST)\\$(APP_NAME)-$(VERSION)-windows-amd64-installer.exe")) { throw "Inno Setup completed without creating the expected installer" }'
	@echo "==> Installer created: $(DIST)/$(APP_NAME)-$(VERSION)-windows-amd64-installer.exe"
else
inno: dist-windows-amd64 _inno-language
	@echo "==> Building Inno Setup installer..."
	@BINARY_PATH="$$(command -v cygpath >/dev/null 2>&1 && cygpath -w "$(PWD)/$(DIST)/$(APP_NAME)-$(VERSION)-windows.exe" || printf '%s' "$(PWD)/$(DIST)/$(APP_NAME)-$(VERSION)-windows.exe")"; \
	TYPST_PATH="$$(command -v cygpath >/dev/null 2>&1 && cygpath -w "$(PWD)/$(DIST)/typst.exe" || printf '%s' "$(PWD)/$(DIST)/typst.exe")"; \
	TINYMIST_PATH="$$(command -v cygpath >/dev/null 2>&1 && cygpath -w "$(PWD)/$(DIST)/tinymist.exe" || printf '%s' "$(PWD)/$(DIST)/tinymist.exe")"; \
	VC_REDIST_PATH="$$(command -v cygpath >/dev/null 2>&1 && cygpath -w "$(PWD)/$(DIST)/vc_redist.x64.exe" || printf '%s' "$(PWD)/$(DIST)/vc_redist.x64.exe")"; \
	OUTPUT_DIR="$$(command -v cygpath >/dev/null 2>&1 && cygpath -w "$(PWD)/$(DIST)" || printf '%s' "$(PWD)/$(DIST)")"; \
	$(INNO_COMPILER) \
		"/DARG_VERSION=\"$(VERSION)\"" \
		"/DARG_FILE_VERSION=\"$(VERSION_BASE).0\"" \
		"/DARG_ARCH=\"amd64\"" \
		"/DARG_BINARY=\"$$BINARY_PATH\"" \
		"/DARG_TYPST_BINARY=\"$$TYPST_PATH\"" \
		"/DARG_TINYMIST_BINARY=\"$$TINYMIST_PATH\"" \
		"/DARG_VC_REDIST=\"$$VC_REDIST_PATH\"" \
		"/DARG_OUTPUT_DIR=\"$$OUTPUT_DIR\"" \
		"/DARG_OUTPUT_BASENAME=\"$(APP_NAME)-$(VERSION)-windows-amd64-installer\"" \
		build/windows/installer/presto.iss
	@echo "==> Installer created: $(DIST)/$(APP_NAME)-$(VERSION)-windows-amd64-installer.exe"
endif

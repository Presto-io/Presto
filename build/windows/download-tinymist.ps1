param(
    [string]$TinymistOut,
    [string]$CacheDir,
    [string]$Archive,
    [string]$BaseUrl,
    [string]$Sha256,
    [string]$Version
)

# Create output directory
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $TinymistOut) | Out-Null

# Download if needed
if (-not (Test-Path $TinymistOut)) {
    if ($Sha256 -eq "") {
     throw "ERROR: TINYMIST_SHA256 is required"
    }

    $cache = Join-Path $CacheDir $Archive
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $cache) | Out-Null

    if (-not (Test-Path $cache)) {
        Write-Host "==> Downloading tinymist $Version ($Archive)..."
        & curl.exe -fL --retry 5 --retry-delay 2 --connect-timeout 30 --max-time 600 "$BaseUrl/$Archive" -o $cache
        if ($LASTEXITCODE -ne 0) {
            Remove-Item -LiteralPath $cache -Force -ErrorAction SilentlyContinue
            exit $LASTEXITCODE
        }
    } else {
        Write-Host "==> Using cached tinymist archive $cache"
    }

    # Verify checksum
    $hash = (Get-FileHash -Algorithm SHA256 $cache).Hash.ToLowerInvariant()
    if ($hash -ne $Sha256) {
        Remove-Item -LiteralPath $cache -Force -ErrorAction SilentlyContinue
        throw "ERROR: tinymist checksum verification failed!"
    }

    # Extract
    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $tmp | Out-Null

    try {
        if ($Archive -like "*.zip") {
            Expand-Archive -LiteralPath $cache -DestinationPath $tmp -Force
        } else {
          & "$env:SystemRoot\System32\tar.exe" -xf $cache -C $tmp
            if ($LASTEXITCODE -ne 0) {
           exit $LASTEXITCODE
          }
        }

        $bin = Get-ChildItem -LiteralPath $tmp -Recurse -File |
          Where-Object { $_.Name -eq "tinymist.exe" -or $_.Name -eq "tinymist" } |
         Sort-Object @{Expression = { if ($_.Name -eq "tinymist.exe") { 0 } else { 1 } }}, @{Expression = "Length"; Descending = $true} |
         Select-Object -First 1

        if (-not $bin) {
            throw "tinymist binary not found in archive"
        }

      Copy-Item -LiteralPath $bin.FullName -Destination $TinymistOut -Force

        $outSig = -join ([System.IO.File]::ReadAllBytes($TinymistOut)[0..1] | ForEach-Object { [char]$_ })
        if ($TinymistOut -like "*.exe" -and $outSig -ne "MZ") {
            Remove-Item -LiteralPath $TinymistOut -Force
            throw "extracted tinymist is not a Windows executable"
        }

        Write-Host "==> $TinymistOut"
    } finally {
      Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

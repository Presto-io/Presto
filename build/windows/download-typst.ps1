param(
    [string]$TypestOut,
    [string]$CacheDir,
    [string]$Archive,
    [string]$BaseUrl,
    [string]$Sha256,
    [string]$RequireSha256,
    [string]$Version
)

# Create output directory
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $TypestOut) | Out-Null

# Recovery logic: check if output exists and is valid
$cache = Join-Path $CacheDir $Archive
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $cache) | Out-Null

if (Test-Path $TypestOut) {
    $sig = -join ([System.IO.File]::ReadAllBytes($TypestOut)[0..1] | ForEach-Object { [char]$_ })
    if ($sig -ne "MZ") {
        if ($Archive -like "*.zip" -and $sig -eq "PK" -and -not (Test-Path $cache)) {
            Move-Item -LiteralPath $TypestOut -Destination $cache -Force
      Write-Host "==> Recovered cached typst archive $cache"
        } else {
            Remove-Item -LiteralPath $TypestOut -Force
        }
    }
}

# Download if needed
if (-not (Test-Path $TypestOut)) {
    if ($RequireSha256 -eq "1" -and $Sha256 -eq "") {
        throw "ERROR: TYPST_SHA256 is required"
    }

    if (-not (Test-Path $cache)) {
        Write-Host "==> Downloading typst $Version ($Archive)..."
        & curl.exe -fL --retry 5 --retry-delay 2 --connect-timeout 30 --max-time 600 "$BaseUrl/$Archive" -o $cache
      if ($LASTEXITCODE -ne 0) {
            Remove-Item -LiteralPath $cache -Force -ErrorAction SilentlyContinue
            exit $LASTEXITCODE
        }
  } else {
     Write-Host "==> Using cached typst archive $cache"
    }

    # Verify checksum
    if ($Sha256 -ne "") {
        $hash = (Get-FileHash -Algorithm SHA256 $cache).Hash.ToLowerInvariant()
        if ($hash -ne $Sha256) {
         Remove-Item -LiteralPath $cache -Force -ErrorAction SilentlyContinue
            throw "ERROR: typst checksum verification failed!"
        }
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
            Where-Object { $_.Name -eq "typst.exe" -or $_.Name -eq "typst" } |
               Sort-Object @{Expression = { if ($_.Name -eq "typst.exe") { 0 } else { 1 } }}, @{Expression = "Length"; Descending = $true} |
               Select-Object -First 1

        if (-not $bin) {
            throw "typst binary not found in archive"
        }

        Copy-Item -LiteralPath $bin.FullName -Destination $TypestOut -Force

        $outSig = -join ([System.IO.File]::ReadAllBytes($TypestOut)[0..1] | ForEach-Object { [char]$_ })
        if ($TypestOut -like "*.exe" -and $outSig -ne "MZ") {
            Remove-Item -LiteralPath $TypestOut -Force
          throw "extracted typst is not a Windows executable"
        }

        Write-Host "==> $TypestOut"
    } finally {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

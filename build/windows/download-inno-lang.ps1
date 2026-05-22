param(
    [string]$LangDir,
    [string]$ZhFile,
    [string]$ZhUrl,
    [string]$ZhSha256
)

# Create language directory
New-Item -ItemType Directory -Force -Path $LangDir | Out-Null

# Download if needed
if (-not (Test-Path $ZhFile)) {
    Write-Host "==> Downloading Inno Setup Simplified Chinese language file..."
    Invoke-WebRequest -UseBasicParsing -Uri $ZhUrl -OutFile "$ZhFile.tmp"

    $hash = (Get-FileHash -Algorithm SHA256 "$ZhFile.tmp").Hash.ToLowerInvariant()
    if ($hash -ne $ZhSha256) {
        Remove-Item -Force "$ZhFile.tmp" -ErrorAction SilentlyContinue
        throw "ERROR: checksum mismatch for $ZhFile"
    }

    Move-Item -LiteralPath "$ZhFile.tmp" -Destination $ZhFile -Force
}

# Verify existing file
$hash = (Get-FileHash -Algorithm SHA256 $ZhFile).Hash.ToLowerInvariant()
if ($hash -ne $ZhSha256) {
    throw "ERROR: checksum mismatch for $ZhFile"
}

Write-Host "==> Language file verified: $ZhFile"

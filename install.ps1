# Installs the prebuilt conjecture binary for Windows from this
# project's GitHub Releases, after checking its SHA-256 against the
# one published next to it.
#
#   irm https://raw.githubusercontent.com/farlleyferreira/conjecture-ai/main/install.ps1 | iex
#
# Environment:
#   CONJECTURE_VERSION      a release tag (e.g. v1.3.0); default: latest
#   CONJECTURE_INSTALL_DIR  where the binary goes;
#                           default: %LOCALAPPDATA%\conjecture\bin
$ErrorActionPreference = 'Stop'

$repo = 'farlleyferreira/conjecture-ai'
$version = if ($env:CONJECTURE_VERSION) { $env:CONJECTURE_VERSION } else { 'latest' }
$installDir = if ($env:CONJECTURE_INSTALL_DIR) { $env:CONJECTURE_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'conjecture\bin' }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    default { throw "conjecture install: unsupported architecture $($env:PROCESSOR_ARCHITECTURE)" }
}

$asset = "conjecture_windows_$arch.zip"
$base = if ($version -eq 'latest') {
    "https://github.com/$repo/releases/latest/download"
} else {
    "https://github.com/$repo/releases/download/$version"
}

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("conjecture-install-" + [System.Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading $asset ($version)..."
    $zip = Join-Path $tmp $asset
    Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset" -OutFile $zip
    Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset.sha256" -OutFile "$zip.sha256"

    $want = ((Get-Content "$zip.sha256" -Raw).Trim() -split '\s+')[0].ToLower()
    $got = (Get-FileHash -Algorithm SHA256 -Path $zip).Hash.ToLower()
    if ($want -ne $got) {
        throw "conjecture install: checksum mismatch for ${asset}: want $want, got $got"
    }

    Expand-Archive -Path $zip -DestinationPath (Join-Path $tmp 'pkg') -Force
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    Copy-Item -Force (Join-Path $tmp 'pkg\conjecture.exe') (Join-Path $installDir 'conjecture.exe')
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

$exe = Join-Path $installDir 'conjecture.exe'
Write-Host "Installed: $exe ($(& $exe version))"

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not (($userPath -split ';') -contains $installDir)) {
    [Environment]::SetEnvironmentVariable('Path', (($userPath.TrimEnd(';') + ';' + $installDir).TrimStart(';')), 'User')
    Write-Host "Added $installDir to your user PATH - open a new terminal to run 'conjecture' directly."
}

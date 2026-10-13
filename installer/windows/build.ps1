# Builds dist\moly-setup-<version>.exe on Windows. Needs: Go, a C compiler for SQLCipher (MSYS2 mingw-w64: gcc on PATH), Inno Setup 6 (iscc).
$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..\..')
$version = (Get-Content (Join-Path $root 'installer\VERSION') -Raw).Trim()
$ids = (Get-Content (Join-Path $root 'installer\extension-ids.txt') | Where-Object { $_ -and -not $_.TrimStart().StartsWith('#') } | ForEach-Object { $_.Trim() }) -join ','
if (-not $ids) { throw 'installer\extension-ids.txt has no extension ID.' }

$out = Join-Path $root 'dist'
New-Item -ItemType Directory -Force $out | Out-Null
$exe = Join-Path $out 'moly.exe'
Push-Location (Join-Path $root 'moly-go')
$env:CGO_ENABLED = '1'; $env:GOOS = 'windows'; $env:GOARCH = 'amd64'
go build -trimpath -ldflags "-s -w -X main.version=$version" -o $exe .
Pop-Location

$iscc = (Get-Command iscc -ErrorAction SilentlyContinue)
if (-not $iscc) { $iscc = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe" } else { $iscc = $iscc.Source }
& $iscc "/DAppVersion=$version" "/DExtensionIds=$ids" "/DSourceExe=$exe" (Join-Path $PSScriptRoot 'moly.iss')
if ($LASTEXITCODE -ne 0) { throw 'Inno Setup failed.' }

$setup = Join-Path $out "moly-setup-$version.exe"
(Get-FileHash $setup -Algorithm SHA256).Hash.ToLower() + "  moly-setup-$version.exe" | Set-Content "$setup.sha256"
Write-Host $setup

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
$env:CGO_ENABLED = '0'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
New-Item -ItemType Directory -Force dist | Out-Null
go build -buildvcs=false -trimpath -ldflags '-s -w -H=windowsgui' -o dist/pCOM-8bit.exe .
if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
Write-Host 'Built dist/pCOM-8bit.exe'

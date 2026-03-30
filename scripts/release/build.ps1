$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$Dist = Join-Path $Root "dist"
$Version = if ($env:VERSION) { $env:VERSION } else { "dev" }
$Commit = if ($env:COMMIT) { $env:COMMIT } else { "unknown" }

if (Test-Path $Dist) {
  Remove-Item $Dist -Recurse -Force
}
New-Item -ItemType Directory -Path $Dist | Out-Null

$Targets = @(
  @{ GOOS = "darwin"; GOARCH = "arm64" },
  @{ GOOS = "darwin"; GOARCH = "amd64" },
  @{ GOOS = "linux"; GOARCH = "amd64" },
  @{ GOOS = "linux"; GOARCH = "arm64" },
  @{ GOOS = "windows"; GOARCH = "amd64" },
  @{ GOOS = "windows"; GOARCH = "arm64" }
)

foreach ($Target in $Targets) {
  $Goos = $Target.GOOS
  $Goarch = $Target.GOARCH
  $OutDir = Join-Path $Dist "codex-proxy_${Version}_${Goos}_${Goarch}"
  New-Item -ItemType Directory -Path $OutDir | Out-Null

  $Bin = if ($Goos -eq "windows") { "codex-proxy.exe" } else { "codex-proxy" }
  Write-Host "==> Building $Goos/$Goarch"
  $env:CGO_ENABLED = "0"
  $env:GOOS = $Goos
  $env:GOARCH = $Goarch
  go build -ldflags "-X main.version=$Version -X main.commit=$Commit" -o (Join-Path $OutDir $Bin) ./cmd/codex-proxy

  Copy-Item (Join-Path $Root "README.md") (Join-Path $OutDir "README.md")
  Copy-Item (Join-Path $Root "README.zh-CN.md") (Join-Path $OutDir "README.zh-CN.md")
  Copy-Item (Join-Path $Root "LICENSE") (Join-Path $OutDir "LICENSE")
  Copy-Item (Join-Path $Root "CODEX_PROXY_FIX_IMPLEMENTATION_BLUEPRINT.md") (Join-Path $OutDir "CODEX_PROXY_FIX_IMPLEMENTATION_BLUEPRINT.md")
  Copy-Item (Join-Path $Root "RELEASE_READINESS_PLAN.md") (Join-Path $OutDir "RELEASE_READINESS_PLAN.md")

  if ($Goos -eq "windows") {
    Compress-Archive -Path $OutDir -DestinationPath (Join-Path $Dist "codex-proxy_${Version}_${Goos}_${Goarch}.zip")
  } else {
    tar -C $Dist -czf (Join-Path $Dist "codex-proxy_${Version}_${Goos}_${Goarch}.tar.gz") (Split-Path $OutDir -Leaf)
  }

  Remove-Item $OutDir -Recurse -Force
}

Get-ChildItem $Dist -File | Where-Object { $_.Name -ne "checksums.txt" } | Get-FileHash -Algorithm SHA256 | ForEach-Object {
  "$($_.Hash.ToLower())  $($_.Path | Split-Path -Leaf)"
} | Set-Content (Join-Path $Dist "checksums.txt")

Write-Host "Artifacts written to $Dist"

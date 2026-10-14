param([switch]$Clients)
$ErrorActionPreference = 'Stop'
$previousGoOS, $previousGoArch, $previousCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
try {
    New-Item -ItemType Directory -Force -Path dist | Out-Null
    $env:CGO_ENABLED = '0'
    $env:GOOS, $env:GOARCH = 'windows', 'amd64'
    go build -trimpath -ldflags '-s -w' -o dist/aoikane.exe ./cmd/aoikane
    if ($LASTEXITCODE) { throw 'Build failed' }
    if ($Clients) {
        foreach ($target in @('linux/amd64', 'linux/arm64', 'darwin/amd64', 'darwin/arm64')) {
            $env:GOOS, $env:GOARCH = $target.Split('/')
            go build -trimpath -ldflags '-s -w' -o "dist/aoikane-$($env:GOOS)-$($env:GOARCH)" ./cmd/aoikane
            if ($LASTEXITCODE) { throw "Build failed: $target" }
        }
    }
} finally {
    $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = $previousGoOS, $previousGoArch, $previousCGO
}

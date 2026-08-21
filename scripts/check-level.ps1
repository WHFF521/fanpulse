param(
    [Parameter(Mandatory = $true)]
    [ValidateRange(0, 14)]
    [int]$Level
)

$ErrorActionPreference = 'Stop'
$env:GOCACHE = Join-Path (Get-Location) '.cache/go-build'

$commands = @{
    0  = @('go test ./coursechecks -run TestLevel00RepositoryBaseline -v')
    1  = @('go test ./internal/platform/config -v')
    2  = @('go test ./internal/identity -v')
    3  = @('go test -race ./internal/event -v')
    4  = @('go test ./internal/httpapi -v', 'go test ./coursechecks -run TestLevel04 -v')
    5  = @('go test ./coursechecks -run TestLevel05 -v', 'go test -tags=integration ./tests/integration -run Postgres -v')
    6  = @('go test -race ./internal/reward -v', 'go test -tags=integration ./tests/integration -run Reward -v')
    7  = @('go test -race ./internal/idempotency -v')
    8  = @('go test -race ./internal/ratelimit -v', 'go test ./coursechecks -run TestLevel08 -v')
    9  = @('go test ./internal/messaging -v', 'go test ./coursechecks -run TestLevel09 -v')
    10 = @('go test -race ./internal/consumer ./internal/worker -v')
    11 = @('go test ./internal/lottery -v')
    12 = @('go test -race ./internal/platform/health -v', 'go test ./coursechecks -run TestLevel12 -v')
    13 = @('go test ./coursechecks -run TestLevel13 -v')
    14 = @('go test -race ./...', 'go test ./coursechecks -run TestLevel14 -v')
}

foreach ($command in $commands[$Level]) {
    Write-Host "> $command" -ForegroundColor Cyan
    Invoke-Expression $command
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Write-Host "Level $Level checks passed." -ForegroundColor Green

# Create an annotated vX.Y.Z tag on main only. Does not push.
param(
    [Parameter(Mandatory = $true)]
    [string]$Tag
)

$ErrorActionPreference = "Stop"
if ($Tag -notmatch '^v\d+\.\d+\.\d+$') {
    Write-Error "TAG must be vX.Y.Z"
    exit 2
}
$branch = git rev-parse --abbrev-ref HEAD
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
if ($branch -ne "main") {
    Write-Error "checkout main first (on $branch)"
    exit 2
}
git tag -a $Tag -m "squad-oc $Tag"
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
Write-Host "created $Tag; publish with: task release:push TAG=$Tag"

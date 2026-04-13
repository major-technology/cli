# Find major CLI - check common install locations
$Major = $null
$Candidates = @(
    (Join-Path $env:USERPROFILE '.major\bin\major.exe'),
    (Join-Path $env:USERPROFILE 'go\bin\major.exe')
)

foreach ($p in $Candidates) {
    if (Test-Path $p) {
        $Major = $p
        break
    }
}

if (-not $Major) {
    $Major = Get-Command major -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source
}

if (-not $Major) { exit 1 }

$Token = & $Major user token 2>$null
$Org   = & $Major org id 2>$null

if (-not $Token -or -not $Org) { exit 1 }

Write-Output "{`"Authorization`": `"Bearer $Token`", `"x-major-org-id`": `"$Org`"}"

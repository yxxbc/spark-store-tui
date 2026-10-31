param(
    [string]$Version = 'v0.9.0',
    [string]$AssetDirectory = 'dist'
)

$ErrorActionPreference = 'Stop'

$token = $env:GITEE_ACCESS_TOKEN
if ([string]::IsNullOrWhiteSpace($token)) {
    $tokenPath = Join-Path $env:USERPROFILE '.sparkstore\gitee-access-token'
    if (Test-Path -LiteralPath $tokenPath) {
        $token = (Get-Content -LiteralPath $tokenPath -Raw).Trim()
    }
}
if ([string]::IsNullOrWhiteSpace($token)) {
    throw 'Set GITEE_ACCESS_TOKEN or create %USERPROFILE%\.sparkstore\gitee-access-token.'
}
if (-not (Test-Path -LiteralPath $AssetDirectory)) {
    throw "Asset directory not found: $AssetDirectory"
}

$owner = 'spark-store-project'
$repository = 'spark-store-tui'
$headers = @{ Authorization = "Bearer $token" }
$tagURL = "https://gitee.com/api/v5/repos/$owner/$repository/releases/tags/$Version"

try {
    $release = Invoke-RestMethod -Method Get -Uri $tagURL -Headers $headers
} catch {
    $release = $null
}
if ($null -eq $release -or -not $release.id) {
    $body = @{
        tag_name         = $Version
        name             = "spark-store-tui $Version"
        target_commitish = 'master'
        body             = 'Spark Store TUI native Linux release.'
    } | ConvertTo-Json
    $release = Invoke-RestMethod -Method Post -Uri "https://gitee.com/api/v5/repos/$owner/$repository/releases" -Headers $headers -ContentType 'application/json' -Body $body
}

Get-ChildItem -LiteralPath $AssetDirectory -File | ForEach-Object {
    & curl.exe --silent --show-error --fail-with-body -X POST -H "Authorization: Bearer $token" -F "file=@$($_.FullName)" "https://gitee.com/api/v5/repos/$owner/$repository/releases/$($release.id)/attach_files" | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to upload $($_.Name)."
    }
    Write-Host "Uploaded $($_.Name)"
}

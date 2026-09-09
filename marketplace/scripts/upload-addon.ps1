# Uploads a marketplace item: creates a GitHub Release with the pack file
# and updates index.json in the repo. Requires the GitHub CLI (gh) installed
# and authenticated (gh auth login).
#
# Example:
#   pwsh scripts/upload-addon.ps1 -Id example-addon -Type mcaddon -Version 1.0.0 `
#     -File .\ExampleAddon-1.0.0.mcaddon -Repo "YOUR_USER/marketplace" `
#     -Name "Example Addon" -Summary "One-line summary" -Icon .\icon.png

param(
    [Parameter(Mandatory = $true)][string]$Id,        # item id (letters/digits/dashes)
    [Parameter(Mandatory = $true)][string]$Type,      # mcaddon | mcpack | mcworld | skinpack
    [Parameter(Mandatory = $true)][string]$Version,   # e.g. 1.0.0
    [Parameter(Mandatory = $true)][string]$File,      # path to the pack file to upload
    [Parameter(Mandatory = $true)][string]$Repo,      # "user/repo" of the marketplace repo
    [string]$Name = "",                               # display name (defaults to $Id)
    [string]$Summary = "",                            # one-line summary
    [string]$DescriptionFile = "",                    # path to a markdown description
    [string]$Icon = "",                               # path to icon.png
    [string]$Screenshots = "",                        # comma-separated paths of screenshots
    [string]$Changelog = "",                          # version changelog text
    [string]$Branch = "main",
    [string]$WorkDir = "."                            # checkout of the marketplace repo
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    Write-Error "GitHub CLI (gh) not found. Install it and run 'gh auth login' first."
}

if (-not (Test-Path $File)) { Write-Error "File not found: $File" }

$fileName  = [IO.Path]::GetFileName($File)
$itemName  = if ($Name) { $Name } else { $Id }
$tag       = "$Id-v$Version"
$jsdelivr  = "https://cdn.jsdelivr.net/gh/$Repo@$Branch"
$rawBase   = "https://github.com/$Repo/releases/download/$tag"

Push-Location $WorkDir
try {
    # 1) Upload the pack file as a GitHub Release
    Write-Host "==> Creating release $tag ..." -ForegroundColor Cyan
    $notes = if ($Changelog) { $Changelog } else { "Release v$Version of $itemName" }
    gh release create $tag $File --repo $Repo --title $tag --notes $notes
    $downloadUrl = "$rawBase/$fileName"
    Write-Host "    downloadUrl: $downloadUrl"

    # 2) Copy icon / screenshots into assets/<id>/
    $assetDir = Join-Path "assets" $Id
    if ($Icon) {
        New-Item -ItemType Directory -Force -Path $assetDir | Out-Null
        Copy-Item $Icon (Join-Path $assetDir "icon.png") -Force
    }
    if ($Screenshots) {
        $i = 1
        foreach ($shot in $Screenshots.Split(",")) {
            $shot = $shot.Trim()
            if (-not $shot) { continue }
            if (-not (Test-Path $shot)) { Write-Warning "screenshot not found: $shot"; continue }
            New-Item -ItemType Directory -Force -Path $assetDir | Out-Null
            Copy-Item $shot (Join-Path $assetDir "shot$i.png") -Force
            $i++
        }
    }

    # 3) Update index.json
    $indexPath = "index.json"
    if (-not (Test-Path $indexPath)) {
        '{ "schemaVersion": 1, "updatedAt": "", "items": [] }' | Set-Content $indexPath -Encoding UTF8
    }
    $index = Get-Content $indexPath -Raw -Encoding UTF8 | ConvertFrom-Json

    $description = if ($DescriptionFile -and (Test-Path $DescriptionFile)) {
        Get-Content $DescriptionFile -Raw -Encoding UTF8
    } else { $Summary }

    $iconUrl = if (Test-Path (Join-Path $assetDir "icon.png")) { "$jsdelivr/assets/$Id/icon.png" } else { "" }
    $shotUrls = @()
    if (Test-Path $assetDir) {
        $shotUrls = Get-ChildItem $assetDir -Filter "shot*.png" |
            ForEach-Object { "$jsdelivr/assets/$Id/$($_.Name)" }
    }

    $newFile = [ordered]@{
        version     = $Version
        date        = (Get-Date -Format "yyyy-MM-dd")
        fileType    = $Type
        fileName    = $fileName
        downloadUrl = $downloadUrl
        size        = (Get-Item $File).Length
        changelog   = $Changelog
    }

    $existing = $index.items | Where-Object { $_.id -eq $Id } | Select-Object -First 1
    if ($existing) {
        $existing.versions = @($existing.versions) + @($newFile)
        if ($Summary)          { $existing.summary     = $Summary }
        if ($description)      { $existing.description = $description }
        if ($iconUrl)          { $existing.icon        = $iconUrl }
        if ($shotUrls.Count)   { $existing.screenshots = @($shotUrls) }
        if (-not ($existing.tags)) { $existing.tags = @() }
    } else {
        $item = [ordered]@{
            id          = $Id
            name        = $itemName
            author      = ""
            summary     = $Summary
            description = $description
            icon        = $iconUrl
            screenshots = @($shotUrls)
            tags        = @()
            type        = $Type
            homepage    = "https://github.com/$Repo"
            versions    = @($newFile)
        }
        $index.items = @($index.items) + @($item)
    }

    $index.updatedAt = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
    ($index | ConvertTo-Json -Depth 10) | Set-Content $indexPath -Encoding UTF8
    Write-Host "==> index.json updated for item '$Id'" -ForegroundColor Green

    Write-Host @"

Next steps (commit and push the repo):
    git add index.json assets/
    git commit -m "marketplace: add $Id v$Version"
    git push origin $Branch
"@ -ForegroundColor Yellow
}
finally {
    Pop-Location
}

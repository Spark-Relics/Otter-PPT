# Otter PPT Font Downloader (PowerShell)
#
# Usage:
#   .\scripts\download_fonts.ps1                    # Google Fonts only
#   .\scripts\download_fonts.ps1 -IncludeSystemFonts # Include Windows CJK fonts
#   .\scripts\download_fonts.ps1 -SystemFontsOnly    # Skip Google Fonts

param(
    [switch]$IncludeSystemFonts,
    [switch]$SystemFontsOnly
)

$ErrorActionPreference = "Continue"

Write-Host "Otter PPT Font Downloader" -ForegroundColor Cyan
Write-Host "=========================" -ForegroundColor Cyan

if ($SystemFontsOnly) {
    $args = @("-no-gstatic", "-sysfonts")
} elseif ($IncludeSystemFonts) {
    $args = @("-sysfonts")
} else {
    $args = @()
}

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Push-Location $projectRoot

try {
    & go run ./cmd/fontdl -dir assets/fonts @args
} finally {
    Pop-Location
}

Write-Host "`nDone!" -ForegroundColor Green

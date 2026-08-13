#!/usr/bin/env bash
# Otter PPT Font Downloader (Bash)
#
# Usage:
#   ./scripts/download_fonts.sh                # Google Fonts only
#   GOFLAGS="-sysfonts" ./scripts/download_fonts.sh  # Include system fonts

set -e

DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$DIR"

echo "Otter PPT Font Downloader"
echo "========================="

FLAGS="${GOFLAGS:-}"
go run ./cmd/fontdl -dir assets/fonts $FLAGS

echo ""
echo "Done!"

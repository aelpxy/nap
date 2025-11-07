#!/bin/bash
set -e

VERSION="${VERSION:-0.1.0-alpha}"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

OUTPUT_DIR="dist"
mkdir -p "$OUTPUT_DIR"


echo "building: $VERSION"

platforms=(
  "linux/amd64"
  "linux/arm64"  
)

for platform in "${platforms[@]}"; do
  IFS='/' read -ra PARTS <<< "$platform"
  GOOS="${PARTS[0]}"
  GOARCH="${PARTS[1]}"

  output_name="yap-${VERSION}-${GOOS}-${GOARCH}"
  if [ "$GOOS" = "windows" ]; then
    output_name+=".exe"
  fi

  echo "  → building $GOOS/$GOARCH..."

  GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="$LDFLAGS" \
    -o "$OUTPUT_DIR/$output_name" \
    .

  file_size=$(du -h "$OUTPUT_DIR/$output_name" | cut -f1)
  echo "    ✓ $output_name ($file_size)"
done

echo "built"

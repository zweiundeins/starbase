#!/bin/sh
# Builds static/vendor/datastar-rocket.js: the Datastar + Rocket release with
# the patches in patches/rocket/ applied, until they land upstream. The
# official release stays next to it (datastar-rocket-$V.js): the install
# snippets point users at jsDelivr's copy and take its integrity from there.
# Usage: DATASTAR_VERSION=v1.0.4 sh scripts/vendor-rocket.sh
set -eu
V="${DATASTAR_VERSION:-v1.0.4}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/static/vendor"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -sSfL -o "$OUT/datastar-rocket-$V.js" "https://cdn.jsdelivr.net/gh/starfederation/datastar@$V/bundles/datastar-rocket.js"
git clone -q --depth 1 --branch "$V" https://github.com/starfederation/datastar.git "$TMP/datastar" 2>/dev/null
git -C "$TMP/datastar" -c user.name=vendor -c user.email=vendor@localhost am -q "$ROOT"/patches/rocket/*.patch

banner="$(head -1 "$OUT/datastar-rocket-$V.js") (patched: patches/rocket)"
cd "$ROOT"
go run github.com/evanw/esbuild/cmd/esbuild "$TMP/datastar/library/src/bundles/datastar-rocket.ts" \
	--bundle --minify --format=esm --target=es2022 --define:ALIAS=null \
	--tsconfig="$TMP/datastar/library/tsconfig.json" \
	--banner:js="$banner" --sourcemap --sources-content=false \
	--outfile="$OUT/datastar-rocket.js" --log-level=warning
echo "wrote $OUT/datastar-rocket.js ($(wc -c < "$OUT/datastar-rocket.js") bytes) from $V + $(ls "$ROOT"/patches/rocket/*.patch | wc -l) patches"

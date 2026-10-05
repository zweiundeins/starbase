#!/bin/sh
# Builds static/vendor/datastar-rocket.js: the Datastar + Rocket release with
# the patches in patches/rocket/ applied, until they land upstream. Pages load
# it, and install snippets pin it (/c/datastar@<hash>/datastar-rocket.js).
# Its declarations go to internal/tscheck/types for the playground's type check.
# Usage: DATASTAR_VERSION=v1.0.4 sh scripts/vendor-rocket.sh
set -eu
V="${DATASTAR_VERSION:-v1.0.4}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/static/vendor"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -sSfL -o "$TMP/official.js" "https://cdn.jsdelivr.net/gh/starfederation/datastar@$V/bundles/datastar-rocket.js"
git clone -q --depth 1 --branch "$V" https://github.com/starfederation/datastar.git "$TMP/datastar" 2>/dev/null
git -C "$TMP/datastar" -c user.name=vendor -c user.email=vendor@localhost am -q "$ROOT"/patches/rocket/*.patch

banner="$(head -1 "$TMP/official.js") (patched: patches/rocket)"
cd "$ROOT"
go run github.com/evanw/esbuild/cmd/esbuild "$TMP/datastar/library/src/bundles/datastar-rocket.ts" \
	--bundle --minify --format=esm --target=es2022 --define:ALIAS=null \
	--tsconfig="$TMP/datastar/library/tsconfig.json" \
	--banner:js="$banner" --sourcemap --sources-content=false \
	--outfile="$OUT/datastar-rocket.js" --log-level=warning
# The map names upstream's files (datastar/library/src/…), not this run's temp dir.
sed 's#"[^"]*/datastar/library/#"datastar/library/#g' "$OUT/datastar-rocket.js.map" > "$TMP/map" && mv "$TMP/map" "$OUT/datastar-rocket.js.map"
echo "wrote $OUT/datastar-rocket.js ($(wc -c < "$OUT/datastar-rocket.js") bytes) from $V + $(ls "$ROOT"/patches/rocket/*.patch | wc -l) patches"

# Declarations of the same source, with the aliases of Datastar's tsconfig
# (@engine, @rocket, ...) made relative, and only what the bundle reaches.
go run ./cmd/fetchtsc -unpack "$TMP/tsc"
L="$TMP/datastar/library"
"$TMP/tsc/tsc" -p "$L/tsconfig.json" --emitDeclarationOnly --rootDir "$L/src" --outDir "$TMP/dts"
for f in "$TMP"/dts/*/*.d.ts "$TMP"/dts/*/*/*.d.ts; do
	up=$(dirname "${f#"$TMP"/dts/}" | sed 's#[^/][^/]*#..#g')
	sed -E "s#([\"'])@engine([\"'])#\1$up/engine/engine\2#g; s#([\"'])@rocket([\"'])#\1$up/rocket/index\2#g; s#([\"'])@(engine|plugins|utils)/#\1$up/\2/#g" "$f" > "$f.new" && mv "$f.new" "$f"
done
echo '{"compilerOptions": {"types": []}, "files": ["bundles/datastar-rocket.d.ts"]}' > "$TMP/dts/tsconfig.json"
rm -rf internal/tscheck/types
"$TMP/tsc/tsc" -p "$TMP/dts/tsconfig.json" --listFilesOnly | while read -r f; do
	case "$f" in "$TMP/dts/"*)
		mkdir -p "internal/tscheck/types/$(dirname "${f#"$TMP"/dts/}")"
		cp "$f" "internal/tscheck/types/${f#"$TMP"/dts/}"
	esac
done
echo "wrote internal/tscheck/types ($(ls -R internal/tscheck/types | grep -c '\.d\.ts$') declaration files)"

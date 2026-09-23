#!/usr/bin/env bash
# Build the visits report for starbase.zweiundeins.gmbh.
#
#   scripts/visits/report.sh [--host libretto] [--out tmp/visits.html] [--keep-json]
#
# collect.py runs on the web host (read-only, via sudo, because the Caddy logs
# are caddy:caddy 640) and prints JSON; only that summary crosses the wire, not
# the ~55 MB of logs. render.py turns it into one self-contained HTML file.
set -euo pipefail

cd "$(dirname "$0")/../.."

HOST="${VISITS_HOST:-libretto}"
OUT=""
KEEP_JSON=0

while [[ $# -gt 0 ]]; do
	case "$1" in
	--host) HOST="$2"; shift 2 ;;
	--out) OUT="$2"; shift 2 ;;
	--keep-json) KEEP_JSON=1; shift ;;
	-h | --help) sed -n '2,8p' "$0" | sed 's/^# \?//'; exit 0 ;;
	*) echo "unknown argument: $1" >&2; exit 2 ;;
	esac
done

: "${OUT:=tmp/visits-$(date +%Y-%m-%d).html}"
mkdir -p "$(dirname "$OUT")"
JSON="${OUT%.html}.json"

echo "Collecting from $HOST ..." >&2
ssh -o BatchMode=yes "$HOST" 'sudo python3 -' <scripts/visits/collect.py >"$JSON"

python3 scripts/visits/render.py <"$JSON" >"$OUT"
[[ $KEEP_JSON -eq 1 ]] || rm -f "$JSON"

echo "Wrote $OUT" >&2

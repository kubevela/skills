#!/usr/bin/env bash
# Tests for kubevela-defkit-author/scripts/emulator-fidelity.py against a small synthetic
# report. Each check prints PASS or FAIL; the script exits 1 if any failed.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
script="$here/../../kubevela-defkit-author/scripts/emulator-fidelity.py"
report="$here/scores.json"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail=0
check() { if [ "$2" = "$3" ]; then echo "PASS $1"; else echo "FAIL $1: expected [$3] got [$2]"; fail=1; fi; }

out="$(python3 "$script" --service dynamodb --file "$report")"
check "reads a report and exits 0" "$?" "0"
check "counts only the service's scenarios, not a service with the same prefix" \
  "$(head -1 <<< "$out" | grep -o '[0-9]* scenarios')" "3 scenarios"
check "counts passed and failed steps per operation" \
  "$(grep '^CreateTable:' <<< "$out")" "CreateTable: 1 passed, 2 did not"
check "names a failing scenario once, with what differed" \
  "$(grep -c 'create-rejected (body)' <<< "$out")" "1"
check "lists every operation when none is named" \
  "$(grep -c '^DeleteTable: 1 passed, 0 did not' <<< "$out")" "1"

out="$(python3 "$script" --service dynamodb --file "$report" --operations DeleteTable)"
check "restricts to the named operations" "$(grep -c '^CreateTable:' <<< "$out")" "0"

python3 "$script" --service dynamodb --file "$report" --operations CreateTable,UpdateTable > "$tmp/out" 2>&1
check "exits 2 when an operation is not in the report" "$?" "2"
check "says which operation is missing" "$(grep -c '^UpdateTable: not in the report' "$tmp/out")" "1"

python3 "$script" --service s3 --file "$report" > "$tmp/out" 2>&1
check "exits 2 for a service that is not in the report" "$?" "2"

python3 "$script" --service dynamodb --file "$here/missing.json" > "$tmp/out" 2>&1
check "exits 2 when the report cannot be read" "$?" "2"

out="$(python3 "$script" --service dynamodb --file "$report" --operations 'CreateTable, DeleteTable')"
check "ignores spaces around operation names" "$(grep -c -E '^(CreateTable|DeleteTable):' <<< "$out")" "2"

echo '[1, 2, 3]' > "$tmp/list.json"
python3 "$script" --service dynamodb --file "$tmp/list.json" > "$tmp/out" 2>&1
check "exits 2 for JSON that is not a report" "$?" "2"
check "says the file is not a report" "$(grep -c 'cannot read the fidelity report' "$tmp/out")" "1"

gzip -c "$report" > "$tmp/scores.json.gz"
check "reads a gzipped report" \
  "$(python3 "$script" --service dynamodb --file "$tmp/scores.json.gz" | grep -c '^CreateTable: 1 passed, 2 did not')" "1"

exit $fail

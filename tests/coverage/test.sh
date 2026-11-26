#!/usr/bin/env bash
# Tests for kubevela-defkit-author/scripts/check-coverage.py. Each bad record is made
# from good.md by one substitution. Each check prints PASS or FAIL; the script exits 1
# if any failed.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
script="$here/../../kubevela-defkit-author/scripts/check-coverage.py"
good="$here/good.md"
cue="$here/definition.cue"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail=0

# expect <name> <exit code> <text the output must contain> <file> [extra args]
expect() {
  local name=$1 code=$2 text=$3 file=$4
  shift 4
  local out rc
  out="$(python3 "$script" "$file" "$@" 2>&1)"
  rc=$?
  if [ "$rc" = "$code" ] && grep -qF -- "$text" <<< "$out"; then
    echo "PASS $name"
  else
    echo "FAIL $name: expected exit $code with [$text], got exit $rc: $out"
    fail=1
  fi
}

# variant <sed expression>: writes a changed copy of good.md and prints its path.
n=0
variant() {
  n=$((n + 1))
  sed -e "$1" "$good" > "$tmp/$n.md"
  echo "$tmp/$n.md"
}

expect "accepts a complete record" 0 "OK 2 verified end to end, 1 endpoint-limited, 1 untested" "$good"
expect "accepts it when every parameter has a row" 0 "OK" "$good" --cue "$cue"

expect "rejects 'not exercised yet' as a reason" 1 '"replicas": untested without a reason' \
  "$(variant 's/needs a second region, and the emulator serves one/not exercised yet/')"
expect "rejects a reason that only starts with the excuse" 1 '"replicas": untested without a reason' \
  "$(variant 's/needs a second region, and the emulator serves one/not tested yet, low priority/')"
expect "rejects TODO as a reason" 1 "untested without a reason" \
  "$(variant 's/needs a second region, and the emulator serves one/TODO/')"
expect "rejects an empty reason" 1 "untested without a reason" \
  "$(variant 's/needs a second region, and the emulator serves one//')"
expect "rejects an endpoint limit without evidence" 1 '"encryption": endpoint-limited without a reason' \
  "$(variant 's/UpdateWidget returns .*emulated`/n\/a/')"
expect "rejects a pass that does not say how it was read back" 1 '"name": verified end to end without a reason' \
  "$(variant 's/| name | verified end to end | DescribeWidget |/| name | verified end to end | |/')"
expect "rejects an outcome outside the three" 1 'outcome "works" is not one of' \
  "$(variant 's/| name | verified end to end |/| name | works |/')"
expect "rejects counts that disagree with the table" 1 "Counts says 3 verified end to end, the table has 2" \
  "$(variant 's/Counts: 2 verified/Counts: 3 verified/')"
expect "rejects a record without a counts line" 1 'no line "Counts:' \
  "$(variant '/^Counts:/d')"
expect "rejects a document without the section" 1 'no "## Runtime verification" section' \
  "$(variant 's/^## Runtime verification/## Notes/')"
expect "rejects a parameter that has no row" 1 'parameter "size" has no row' \
  "$(variant '/^| size, at creation/d;s/Counts: 2 verified/Counts: 1 verified/')" --cue "$cue"
expect "does not take a helper field or a validator block for a parameter" 0 "OK" "$good" --cue "$cue"
expect "rejects a row with more than three cells" 1 '"name": the row has 4 cells, not 3' \
  "$(variant 's/| name | verified end to end | DescribeWidget |/| name | verified end to end | DescribeWidget | extra |/')"
expect "accepts an escaped bar inside a cell" 0 "OK" \
  "$(variant 's/| name | verified end to end | DescribeWidget |/| name | verified end to end | DescribeWidget, value `a\\\\|b` |/')"

# A second, optional "Real-cloud verification" section follows the same rules and may be a subset.
real="$here/good-real.md"
variant_real() {
  n=$((n + 1))
  sed -e "$1" "$real" > "$tmp/$n.md"
  echo "$tmp/$n.md"
}
expect "accepts a real-cloud section next to the emulator record" 0 "OK 2 verified end to end, 1 endpoint-limited, 1 untested; real cloud: 1 verified end to end, 1 endpoint-limited, 1 untested" "$real"
expect "does not require every parameter in the real-cloud section" 0 "OK" "$real" --cue "$cue"
expect "rejects an outcome outside the three in the real-cloud section" 1 'Real-cloud verification: "name": outcome "works" is not one of' \
  "$(variant_real 's/| name | verified end to end | DescribeWidget against the account |/| name | works | DescribeWidget against the account |/')"
expect "rejects real-cloud counts that disagree with its table" 1 "Real-cloud verification: Counts says 2 verified end to end, the table has 1" \
  "$(variant_real 's/Counts: 1 verified end to end, 1 endpoint-limited, 1 untested/Counts: 2 verified end to end, 1 endpoint-limited, 1 untested/')"
expect "rejects a real-cloud untested row without a blocker" 1 'Real-cloud verification: "replicas": untested without a reason' \
  "$(variant_real 's/one extra widget-hour per run; not in the budget/not exercised yet/')"
expect "still reports the emulator table by its own name" 1 'Runtime verification: Counts says 3 verified end to end, the table has 2' \
  "$(variant_real 's/Counts: 2 verified/Counts: 3 verified/')"
expect "rejects a real-cloud lever the emulator table does not name" 1 'Real-cloud verification: "sharding" is not in the Runtime verification table' \
  "$(variant_real 's/| name | verified end to end | DescribeWidget against the account |/| sharding | verified end to end | DescribeWidget against the account |/')"

# A parameter block at column 0 after a blank line: the indent must not swallow the newline.
printf 'widget: type: "component"\n\nparameter: {\n\tname!: string\n\tmissing?: int\n}\n' > "$tmp/flat.cue"
expect "reads a parameter block at column 0" 1 'parameter "missing" has no row' "$good" --cue "$tmp/flat.cue"

exit $fail

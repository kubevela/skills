#!/usr/bin/env bash
# Tests for kubevela-platform-engineer/scripts/definition-consumers.sh against a synthetic
# Application list. Each check prints PASS or FAIL; the script exits 1 if any failed.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
script="$here/../../kubevela-platform-engineer/scripts/definition-consumers.sh"
fixture="$here/fixture.json"
fail=0
check() { if [ "$2" = "$3" ]; then echo "PASS $1"; else echo "FAIL $1: expected [$3] got [$2]"; fail=1; fi; }

out="$(bash "$script" webservice --from-file "$fixture" --json)"

check "deduplicates an Application with two matching components" \
  "$(jq -r '[.consumers[] | select(.app=="two-components")] | length' <<< "$out")" "1"
check "discovers six distinct consumers" \
  "$(jq -r '.summary.discovered' <<< "$out")" "6"
check "finds the revision-pinned consumer" \
  "$(jq -r '[.consumers[] | select(.app=="pinned")] | length' <<< "$out")" "1"
check "pinned-only consumer is not affected" \
  "$(jq -r '.consumers[] | select(.app=="pinned") | .affected' <<< "$out")" "false"
check "finds a consumer through a nested trait" \
  "$(jq -r '.consumers[] | select(.app=="trait-user") | .refs[0].kind' <<< "$out")" "trait"
check "finds a consumer through a policy" \
  "$(jq -r '.consumers[] | select(.app=="policy-user") | .refs[0].kind' <<< "$out")" "policy"
check "finds a consumer through a workflow sub-step" \
  "$(jq -r '.consumers[] | select(.app=="step-user") | .refs[0].kind' <<< "$out")" "workflow-step"
check "mixed pinned and floating counts as affected" \
  "$(jq -r '.consumers[] | select(.app=="mixed") | .affected' <<< "$out")" "true"
check "affected total excludes pinned-only" \
  "$(jq -r '.summary.affected' <<< "$out")" "5"
check "does not match a different definition with a shared prefix" \
  "$(jq -r '[.consumers[] | select(.app=="unrelated")] | length' <<< "$out")" "0"
check "tolerates an Application with no spec" \
  "$(jq -r '[.consumers[] | select(.app=="no-spec")] | length' <<< "$out")" "0"
check "--kind trait restricts to trait references" \
  "$(bash "$script" webservice --from-file "$fixture" --kind trait --json | jq -r '.summary.discovered')" "1"
check "reports complete visibility when every namespace was read" \
  "$(jq -r '.summary.complete' <<< "$out")" "true"

exit $fail

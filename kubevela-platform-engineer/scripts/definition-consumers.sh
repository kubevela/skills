#!/usr/bin/env bash
# Inventory the Applications that reference a KubeVela definition.
#
#   definition-consumers.sh <definition-name> [--kind component|trait|policy|workflow-step]
#                           [--namespaces ns1,ns2] [--from-file apps.json] [--json]
#
# Walks components, nested traits, policies and workflow steps (including sub-steps),
# understands revision-qualified references such as `webservice@v3`, deduplicates
# Applications, and separates references that float on the latest revision (affected by
# a change to the definition) from references pinned to a revision (not affected).
#
# Visibility is reported, not assumed: namespaces that could not be listed are named in
# the summary so a partial inventory is never presented as a complete count.
#
# Requires kubectl and jq. With --from-file the Application list is read from a JSON file
# (the output of `kubectl get applications -o json`) instead of the cluster, which is how
# the tests run.
set -euo pipefail

name=""
kind=""
namespaces=""
from_file=""
as_json=0

while [ $# -gt 0 ]; do
  case "$1" in
    --kind) kind="$2"; shift 2 ;;
    --namespaces) namespaces="$2"; shift 2 ;;
    --from-file) from_file="$2"; shift 2 ;;
    --json) as_json=1; shift ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) if [ -z "$name" ]; then name="$1"; shift; else echo "unexpected argument: $1" >&2; exit 2; fi ;;
  esac
done

if [ -z "$name" ]; then
  echo "usage: definition-consumers.sh <definition-name> [--kind ...] [--namespaces ...] [--from-file ...] [--json]" >&2
  exit 2
fi
case "$kind" in ""|component|trait|policy|workflow-step) ;; *) echo "unknown --kind: $kind" >&2; exit 2 ;; esac

# Collect Applications. Each item records the namespace it came from so unreadable
# namespaces can be reported.
unreadable=()
if [ -n "$from_file" ]; then
  apps="$(jq -c '.items // []' "$from_file")"
elif [ -n "$namespaces" ]; then
  apps="[]"
  IFS=',' read -r -a ns_list <<< "$namespaces"
  for ns in "${ns_list[@]}"; do
    if out="$(kubectl get applications -n "$ns" -o json 2>/dev/null)"; then
      apps="$(jq -c --argjson a "$apps" '$a + (.items // [])' <<< "$out")"
    else
      unreadable+=("$ns")
    fi
  done
else
  if ! apps="$(kubectl get applications -A -o json 2>/dev/null | jq -c '.items // []')"; then
    echo "cannot list applications cluster-wide; pass --namespaces with the namespaces you can read" >&2
    exit 1
  fi
fi

# One row per reference, then collapse to one row per Application.
rows="$(jq -c --arg name "$name" --arg kind "$kind" '
  def refs:
    . as $app
    | [
        (($app.spec.components // [])[]
          | {kind: "component", type: .type, where: ("component/" + (.name // "?"))}),
        (($app.spec.components // [])[]
          | . as $c
          | (.traits // [])[]
          | {kind: "trait", type: .type, where: ("component/" + ($c.name // "?") + "/trait")}),
        (($app.spec.policies // [])[]
          | {kind: "policy", type: .type, where: ("policy/" + (.name // "?"))}),
        (($app.spec.workflow.steps // [])[]
          | . as $s
          | ({kind: "workflow-step", type: .type, where: ("step/" + (.name // "?"))},
             ((.subSteps // [])[] | {kind: "workflow-step", type: .type, where: ("step/" + ($s.name // "?") + "/sub")})))
      ]
    | map(select(.type != null))
    | map(. + {base: (.type | split("@")[0]), revision: ((.type | split("@")[1]) // null)})
    | map(select(.base == $name))
    | map(select($kind == "" or .kind == $kind));
  .[]
  | {ns: .metadata.namespace, app: .metadata.name, refs: refs}
  | select((.refs | length) > 0)
  | . + {
      pinned:   ([.refs[] | select(.revision != null)] | length),
      floating: ([.refs[] | select(.revision == null)] | length)
    }
  | . + {affected: (.floating > 0)}
' <<< "$apps")"

discovered="$(jq -s 'length' <<< "$rows")"
affected="$(jq -s 'map(select(.affected)) | length' <<< "$rows")"
pinned_only="$(jq -s 'map(select(.affected | not)) | length' <<< "$rows")"

if [ "$as_json" -eq 1 ]; then
  jq -s --arg name "$name" --argjson unreadable "$(printf '%s\n' "${unreadable[@]:-}" | jq -R . | jq -s 'map(select(length>0))')" '
    {definition: $name,
     summary: {discovered: length,
               affected: (map(select(.affected)) | length),
               pinnedOnly: (map(select(.affected | not)) | length),
               unreadableNamespaces: $unreadable,
               complete: ($unreadable | length == 0)},
     consumers: .}' <<< "$rows"
  exit 0
fi

echo "definition: $name"
echo "consumers discovered: $discovered   affected by a change to latest: $affected   pinned only: $pinned_only"
if [ "${#unreadable[@]}" -gt 0 ]; then
  echo "INCOMPLETE: could not list namespaces: ${unreadable[*]}"
else
  echo "visibility: complete for the namespaces queried"
fi
echo
printf '%-40s %-9s %-8s %s\n' "APPLICATION" "AFFECTED" "PINNED" "REFERENCES"
jq -r '
  . as $r
  | ($r.ns + "/" + $r.app) as $id
  | ($r.refs | map(.kind + ":" + .where + (if .revision then "@" + .revision else "" end)) | join(", ")) as $refs
  | [$id, (if $r.affected then "yes" else "no" end), ($r.pinned|tostring), $refs] | @tsv
' <<< "$rows" | awk -F'\t' '{printf "%-40s %-9s %-8s %s\n", $1, $2, $3, $4}'

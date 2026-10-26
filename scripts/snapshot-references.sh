#!/usr/bin/env bash
# Regenerate the version-stamped reference snapshots under each skill from a KubeVela
# checkout at a given tag.
#
#   scripts/snapshot-references.sh <path-to-kubevela-checkout> <tag>
#   e.g. scripts/snapshot-references.sh ../kubevela v1.11.0
#
# Produces:
#   kubevela-developer/references/application-shape.md        Application spec and status
#   kubevela-sre/references/status-shapes.md                   Application and ApplicationRevision status,
#                                                              ResourceTracker and Workflow spec
#   kubevela-platform-engineer/references/definition-shapes.md The five definition CRDs and DefinitionRevision
#
# The tag is recorded in every file's header; the header has no date, so running this
# twice at the same tag is a no-op. Run it when the pinned KubeVela version moves, commit
# the result, and let CI's reference check confirm the stamp matches the version in
# kubevela-sre/references/servers.yaml.
set -euo pipefail
repo="$1"; tag="$2"
here="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
for f in applications applicationrevisions componentdefinitions traitdefinitions policydefinitions workflowstepdefinitions definitionrevisions resourcetrackers workflows; do
  git -C "$repo" show "$tag:charts/vela-core/crds/core.oam.dev_$f.yaml" > "$tmp/$f.yaml"
done
src="kubevela $tag (charts/vela-core/crds)"
gen=(python3 "$here/scripts/snapshot_crd_shape.py" --source "$src")
mkdir -p "$here/kubevela-developer/references" "$here/kubevela-sre/references" "$here/kubevela-platform-engineer/references"
"${gen[@]}" --max-depth 4 "$tmp/applications.yaml" > "$here/kubevela-developer/references/application-shape.md"
"${gen[@]}" --max-depth 4 --only status "$tmp/applications.yaml" "$tmp/applicationrevisions.yaml" > "$here/kubevela-sre/references/status-shapes.md"
"${gen[@]}" --no-header --max-depth 3 --only spec "$tmp/resourcetrackers.yaml" "$tmp/workflows.yaml" >> "$here/kubevela-sre/references/status-shapes.md"
"${gen[@]}" --max-depth 5 "$tmp/componentdefinitions.yaml" "$tmp/traitdefinitions.yaml" "$tmp/policydefinitions.yaml" "$tmp/workflowstepdefinitions.yaml" "$tmp/definitionrevisions.yaml" > "$here/kubevela-platform-engineer/references/definition-shapes.md"
echo "snapshots regenerated from $src"

#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

LOCK_FILE="research/openwaldo-sync-v1/upstream.lock.json"
UPSTREAM_URL="${OPENWALDO_UPSTREAM_URL:-https://github.com/openwaldo/waldo.git}"
UPSTREAM_REF="${1:-${OPENWALDO_UPSTREAM_REF:-main}}"

fail() {
  printf 'openwaldo-refresh: %s\n' "$*" >&2
  exit 1
}

allow_overlay_path() {
  case "$1" in
    AXM_MIRROR_EXPERIMENT.md|\
    THIRD_PARTY.json|\
    cmd/waldo-axm-mirror/*|\
    cmd/waldo-mirror-review/*|\
    internal/axmmirror/*|\
    examples/axm-mirror/*|\
    experiments/*|\
    research/*|\
    docs/adr/9*.md|\
    .github/workflows/axm-*.yml|\
    .github/workflows/axm-*.yaml)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

test -f "$LOCK_FILE" || fail "missing $LOCK_FILE"
git diff --quiet || fail "working tree must be clean before refresh"
git diff --cached --quiet || fail "index must be clean before refresh"

SOURCE_HEAD="$(git rev-parse HEAD)"
PREVIOUS_UPSTREAM="$(
  python3 - "$LOCK_FILE" <<'PY'
import json, sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    print(json.load(f)["commit"])
PY
)"

git fetch --no-tags "$UPSTREAM_URL" "$UPSTREAM_REF"
NEW_UPSTREAM="$(git rev-parse FETCH_HEAD)"
NEW_TREE="$(git rev-parse "$NEW_UPSTREAM^{tree}")"

if ! git cat-file -e "$PREVIOUS_UPSTREAM^{commit}" 2>/dev/null; then
  git fetch --no-tags "$UPSTREAM_URL" "$PREVIOUS_UPSTREAM"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
overlay="$tmpdir/overlay.txt"
unclassified="$tmpdir/unclassified.txt"
collisions="$tmpdir/collisions.txt"
: > "$overlay"
: > "$unclassified"
: > "$collisions"

while IFS= read -r path; do
  if git cat-file -e "$PREVIOUS_UPSTREAM:$path" 2>/dev/null; then
    continue
  fi
  if allow_overlay_path "$path"; then
    printf '%s\n' "$path" >> "$overlay"
  else
    printf '%s\n' "$path" >> "$unclassified"
  fi
done < <(git ls-tree -r --name-only "$SOURCE_HEAD")

if test -s "$unclassified"; then
  echo "Files outside the previous OpenWALDO body are not classified as AXM/Mirror overlay:"
  cat "$unclassified"
  fail "refusing to guess ownership"
fi

while IFS= read -r path; do
  test -n "$path" || continue
  if git cat-file -e "$NEW_UPSTREAM:$path" 2>/dev/null; then
    printf '%s\n' "$path" >> "$collisions"
  fi
done < "$overlay"

if test -s "$collisions"; then
  echo "OpenWALDO now owns path(s) previously owned only by the AXM/Mirror overlay:"
  cat "$collisions"
  fail "ownership collision requires explicit review"
fi

# Start from the exact new OpenWALDO tree, then restore only classified overlay paths.
git read-tree --reset -u "$NEW_UPSTREAM^{tree}"
while IFS= read -r path; do
  test -n "$path" || continue
  git checkout "$SOURCE_HEAD" -- "$path"
done < "$overlay"

if test "$NEW_UPSTREAM" != "$PREVIOUS_UPSTREAM"; then
  python3 - "$LOCK_FILE" "$UPSTREAM_REF" "$NEW_UPSTREAM" "$NEW_TREE" "$PREVIOUS_UPSTREAM" <<'PY'
import json, sys
path, ref, commit, tree, previous = sys.argv[1:]
payload = {
    "schema_version": "axm-openwaldo-upstream-lock.v1",
    "source_repo": "openwaldo/waldo",
    "source_url": "https://github.com/openwaldo/waldo",
    "source_ref": ref,
    "commit": commit,
    "tree": tree,
    "previous_commit": previous,
    "policy": {
        "upstream_owns_existing_paths": True,
        "overlay_is_additive_only": True,
        "path_collisions_fail_closed": True,
        "walmi_promotion_is_never_automatic": True
    }
}
with open(path, "w", encoding="utf-8") as f:
    json.dump(payload, f, indent=2, sort_keys=True)
    f.write("\n")
PY
fi

git add -A
candidate_tree="$(git write-tree)"
failures=0
while IFS=$'\t' read -r meta path; do
  expected_mode="$(awk '{print $1}' <<<"$meta")"
  expected_type="$(awk '{print $2}' <<<"$meta")"
  expected_sha="$(awk '{print $3}' <<<"$meta")"
  actual="$(git ls-tree "$candidate_tree" -- "$path")"
  if test -z "$actual"; then
    echo "MISSING UPSTREAM PATH: $path"
    failures=$((failures + 1))
    continue
  fi
  actual_mode="$(awk '{print $1}' <<<"$actual")"
  actual_type="$(awk '{print $2}' <<<"$actual")"
  actual_sha="$(awk '{print $3}' <<<"$actual")"
  if test "$expected_mode" != "$actual_mode" || \
     test "$expected_type" != "$actual_type" || \
     test "$expected_sha" != "$actual_sha"; then
    echo "UPSTREAM DRIFT: $path"
    failures=$((failures + 1))
  fi
done < <(git ls-tree -r "$NEW_UPSTREAM")

test "$failures" -eq 0 || fail "candidate changed current OpenWALDO bytes or modes"
git diff --check

changed=true
if git diff --cached --quiet; then
  changed=false
fi

printf 'previous_upstream=%s\n' "$PREVIOUS_UPSTREAM"
printf 'upstream_sha=%s\n' "$NEW_UPSTREAM"
printf 'upstream_tree=%s\n' "$NEW_TREE"
printf 'overlay_paths=%s\n' "$(wc -l < "$overlay" | tr -d ' ')"
printf 'changed=%s\n' "$changed"
echo "Current OpenWALDO body in candidate: EXACT"

if test -n "${GITHUB_OUTPUT:-}"; then
  {
    printf 'previous_upstream=%s\n' "$PREVIOUS_UPSTREAM"
    printf 'upstream_sha=%s\n' "$NEW_UPSTREAM"
    printf 'upstream_tree=%s\n' "$NEW_TREE"
    printf 'upstream_short=%s\n' "${NEW_UPSTREAM:0:12}"
    printf 'overlay_paths=%s\n' "$(wc -l < "$overlay" | tr -d ' ')"
    printf 'changed=%s\n' "$changed"
  } >> "$GITHUB_OUTPUT"
fi

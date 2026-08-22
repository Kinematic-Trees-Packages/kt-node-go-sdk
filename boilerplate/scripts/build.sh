#!/usr/bin/env bash
set -euo pipefail
out="${KTM_BUILD_OUTPUT:-build/ktm-output}"
rm -rf "$out"
mkdir -p "$out/compiled/bin" "$out/source"

split_paths() {
  local value=${1:-}
  IFS=':' read -r -a __paths <<<"$value"
  for p in "${__paths[@]}"; do
    [[ -n "$p" ]] && printf '%s\n' "$p"
  done
}

cflags_from_cpath() {
  while IFS= read -r root; do printf ' -I%s' "$root"; done < <(split_paths "${CPATH:-}")
}

ldflags_from_library_path() {
  while IFS= read -r root; do printf ' -L%s' "$root"; done < <(split_paths "${LIBRARY_PATH:-}")
}

kt_node_shared_library() {
  while IFS= read -r root; do
    if [[ -f "$root/libkt_node.so" ]]; then
      printf ' %s' "$root/libkt_node.so"
      return 0
    fi
  done < <(split_paths "${LIBRARY_PATH:-}")
  echo "libkt_node.so not found in LIBRARY_PATH; run through KTM so kt-node libraryPaths are composed" >&2
  return 2
}

export CGO_ENABLED=1
cgo_cflags="${CGO_CFLAGS:-}$(cflags_from_cpath)"
cgo_ldflags="${CGO_LDFLAGS:-}$(ldflags_from_library_path)$(kt_node_shared_library)"
export CGO_CFLAGS="$cgo_cflags"
export CGO_LDFLAGS="$cgo_ldflags"
# shellcheck disable=SC1083 # KTM template placeholder is rendered before execution.
go build -o "$out/compiled/bin/{{KTM_CREATE_PROJECT_NAME}}" ./cmd/{{KTM_CREATE_PROJECT_NAME}}
cp -a README.md package.ktm.json scripts go.mod cmd internal examples "$out/source"/
echo "Built {{KTM_CREATE_PROJECT_NAME}} Go compiled package into $out/compiled"

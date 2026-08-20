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

export CGO_ENABLED=1
export CGO_CFLAGS="${CGO_CFLAGS:-}$(cflags_from_cpath)"
export CGO_LDFLAGS="${CGO_LDFLAGS:-}$(ldflags_from_library_path) -lkt_node"
go build -o "$out/compiled/bin/{{KTM_CREATE_PROJECT_NAME}}" ./cmd/{{KTM_CREATE_PROJECT_NAME}}
cp -a README.md package.ktm.json ktm-pack.json scripts go.mod cmd internal examples "$out/source"/
echo "Built {{KTM_CREATE_PROJECT_NAME}} Go compiled package into $out/compiled"

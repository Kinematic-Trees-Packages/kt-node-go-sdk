#!/usr/bin/env bash
set -euo pipefail

split_paths() {
  local value=${1:-}
  IFS=':' read -r -a __paths <<<"$value"
  for p in "${__paths[@]}"; do
    [[ -n "$p" ]] && printf '%s\n' "$p"
  done
}

find_in_path_list() {
  local env_name=$1
  local rel=$2
  local value=${!env_name:-}
  while IFS= read -r root; do
    [[ -e "$root/$rel" ]] && return 0
  done < <(split_paths "$value")
  return 1
}

require_kt_node_build_env() {
  local missing=0
  if ! find_in_path_list CPATH kt_node.h; then
    echo "kt-node header kt_node.h not found in CPATH; run through KTM so kt-node includePaths are composed" >&2
    missing=1
  fi
  if ! find_in_path_list LIBRARY_PATH libkt_node.so && ! find_in_path_list LIBRARY_PATH libkt_node.a && ! find_in_path_list LD_LIBRARY_PATH libkt_node.so; then
    echo "kt-node library not found in LIBRARY_PATH/LD_LIBRARY_PATH; run through KTM so kt-node libraryPaths are composed" >&2
    missing=1
  fi
  if [[ $missing -ne 0 ]]; then
    exit 2
  fi
}

safe_path_list() {
  local env_name=$1
  local kind=$2
  local value=${!env_name:-}
  local safe_root=".ktm-go-paths/$kind"
  local index=0
  mkdir -p "$safe_root"
  while IFS= read -r root; do
    local safe_path="$safe_root/p$index"
    rm -f "$safe_path"
    ln -s "$root" "$safe_path"
    if [[ $index -gt 0 ]]; then printf ':'; fi
    printf '%s' "$PWD/$safe_path"
    index=$((index + 1))
  done < <(split_paths "$value")
}

cflags_from_cpath() {
  while IFS= read -r root; do printf ' -I%s' "$root"; done < <(split_paths "${CPATH:-}")
}

ldflags_from_library_path() {
  # Keep this intentionally linker-portable. TeamCity's Go toolchain can route
  # cgo through a clang/zig-style linker that rejects GNU ld's -rpath-link, and
  # the smoke already exports LD_LIBRARY_PATH for runtime discovery.
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

require_kt_node_build_env
cpath_value="$(safe_path_list CPATH cpath)"
library_path_value="$(safe_path_list LIBRARY_PATH libpath)"
ld_library_path_value="$(safe_path_list LD_LIBRARY_PATH ldpath)"
export CPATH="$cpath_value"
export LIBRARY_PATH="$library_path_value"
export LD_LIBRARY_PATH="$ld_library_path_value"
export CGO_ENABLED=1
cgo_cflags="${CGO_CFLAGS:-}$(cflags_from_cpath)"
cgo_ldflags="${CGO_LDFLAGS:-}$(ldflags_from_library_path)$(kt_node_shared_library)"
export CGO_CFLAGS="$cgo_cflags"
export CGO_LDFLAGS="$cgo_ldflags"
unset LD_RUN_PATH
go test ./...
# shellcheck disable=SC1083 # KTM template placeholder is rendered before execution.
go run ./cmd/{{KTM_CREATE_PROJECT_NAME}}
echo "Go kt-node smoke passed"

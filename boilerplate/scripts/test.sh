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
  if ! find_in_path_list CPATH kt_robotics.h; then
    echo "kt-node header kt_robotics.h not found in CPATH; run through KTM so kt-node includePaths are composed" >&2
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

require_kt_node_build_env
export CPATH="$(safe_path_list CPATH cpath)"
export LIBRARY_PATH="$(safe_path_list LIBRARY_PATH libpath)"
export LD_LIBRARY_PATH="$(safe_path_list LD_LIBRARY_PATH ldpath)"
export CGO_ENABLED=1
export CGO_CFLAGS="${CGO_CFLAGS:-}$(cflags_from_cpath)"
export CGO_LDFLAGS="${CGO_LDFLAGS:-}$(ldflags_from_library_path) -lkt_node"
go test ./...
go run ./cmd/{{KTM_CREATE_PROJECT_NAME}}
echo "Go kt-node smoke passed"

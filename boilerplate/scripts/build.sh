#!/usr/bin/env bash
set -euo pipefail
out="${KTM_BUILD_OUTPUT:-build/ktm-output}"
rm -rf "$out"
mkdir -p "$out/compiled/bin" "$out/source"
cp -a package.ktm.json runtime.json "$out/"

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

remove_elf_search_paths() {
  local binary=$1
  if command -v patchelf >/dev/null 2>&1; then
    patchelf --remove-rpath "$binary"
    return 0
  fi
  python3 - "$binary" <<'PY'
import struct
import sys

path = sys.argv[1]
with open(path, "r+b") as handle:
    data = bytearray(handle.read())
    if data[:4] != b"\x7fELF":
        raise SystemExit(0)
    elf_class = data[4]
    endian = "<" if data[5] == 1 else ">"
    if elf_class == 2:
        phoff = struct.unpack_from(endian + "Q", data, 32)[0]
        phentsize = struct.unpack_from(endian + "H", data, 54)[0]
        phnum = struct.unpack_from(endian + "H", data, 56)[0]
        dyn_entry_size = 16
        tag_fmt = endian + "q"
        word_fmt = endian + "Q"

        def program_header(index):
            offset = phoff + index * phentsize
            return (
                struct.unpack_from(endian + "I", data, offset)[0],
                struct.unpack_from(endian + "Q", data, offset + 8)[0],
                struct.unpack_from(endian + "Q", data, offset + 32)[0],
            )
    elif elf_class == 1:
        phoff = struct.unpack_from(endian + "I", data, 28)[0]
        phentsize = struct.unpack_from(endian + "H", data, 42)[0]
        phnum = struct.unpack_from(endian + "H", data, 44)[0]
        dyn_entry_size = 8
        tag_fmt = endian + "i"
        word_fmt = endian + "I"

        def program_header(index):
            offset = phoff + index * phentsize
            return (
                struct.unpack_from(endian + "I", data, offset)[0],
                struct.unpack_from(endian + "I", data, offset + 4)[0],
                struct.unpack_from(endian + "I", data, offset + 16)[0],
            )
    else:
        raise SystemExit("unsupported ELF class")

    changed = 0
    for index in range(phnum):
        segment_type, segment_offset, segment_size = program_header(index)
        if segment_type != 2:  # PT_DYNAMIC
            continue
        for entry_offset in range(segment_offset, segment_offset + segment_size, dyn_entry_size):
            tag = struct.unpack_from(tag_fmt, data, entry_offset)[0]
            if tag == 0:  # DT_NULL
                break
            if tag in (15, 29):  # DT_RPATH, DT_RUNPATH
                struct.pack_into(tag_fmt, data, entry_offset, 30)  # DT_FLAGS
                struct.pack_into(word_fmt, data, entry_offset + dyn_entry_size // 2, 0)
                changed += 1
    if changed:
        handle.seek(0)
        handle.write(data)
        handle.truncate()
PY
}

export CGO_ENABLED=1
cgo_cflags="${CGO_CFLAGS:-}$(cflags_from_cpath)"
cgo_ldflags="${CGO_LDFLAGS:-}$(ldflags_from_library_path)$(kt_node_shared_library)"
export CGO_CFLAGS="$cgo_cflags"
export CGO_LDFLAGS="$cgo_ldflags"
unset LD_RUN_PATH
# shellcheck disable=SC1083 # KTM template placeholder is rendered before execution.
go build -o "$out/compiled/bin/{{KTM_CREATE_PROJECT_NAME}}" ./cmd/{{KTM_CREATE_PROJECT_NAME}}
remove_elf_search_paths "$out/compiled/bin/{{KTM_CREATE_PROJECT_NAME}}"
cp -a README.md scripts go.mod cmd internal examples "$out/source"/
echo "Built {{KTM_CREATE_PROJECT_NAME}} Go compiled package into $out/compiled"

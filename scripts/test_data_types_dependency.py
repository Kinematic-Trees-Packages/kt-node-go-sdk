#!/usr/bin/env python3
"""Verify SDK and generated-project manifests lock one kt-messages release."""

import json
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SDK_NAME = "kt-go-sdk"
EXPECTED = {
    "owner": "kinematic-trees",
    "name": "kt-messages",
    "version": "0.1.0",
    "classification": "data_types",
}


def load(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8").replace("{{KTM_CREATE_RUN_ENVIRONMENTS_JSON}}", "[]"))


for relative in ("package.ktm.json", "boilerplate/package.ktm.json.template"):
    manifest = load(ROOT / relative)
    dependencies = manifest["dependencies"]["packages"]
    matches = [item for item in dependencies if item.get("name") == "kt-messages"]
    if len(matches) != 1 or any(matches[0].get(key) != value for key, value in EXPECTED.items()):
        raise SystemExit(f"{relative}: expected one exact kt-messages data_types dependency")
    if set(matches[0].get("environments", {}).values()) != {"portable"}:
        raise SystemExit(f"{relative}: kt-messages must map every parent to portable")
    runtime = [item for item in dependencies if item.get("classification") == "library"]
    expected_version = "0.1.1" if relative == "package.ktm.json" else "{{KTM_CREATE_KT_NODE_VERSION}}"
    if len(runtime) != 1 or (runtime[0].get("owner"), runtime[0].get("name"), runtime[0].get("version")) != (
        "kinematic-trees", "libkt", expected_version
    ):
        raise SystemExit(f"{relative}: expected one exact libkt library dependency")
    if relative.startswith("boilerplate/"):
        if "runtime.json" not in manifest["files"] or "runtime" in manifest["files"]:
            raise SystemExit(f"{relative}: expected a root runtime.json and no runtime directory")
        if "development" in manifest:
            raise SystemExit(f"{relative}: legacy development.commands is forbidden")

runtime = load(ROOT / "boilerplate/runtime.json.template")
if runtime.get("package") != "./package.ktm.json":
    raise SystemExit("boilerplate/runtime.json.template must reference ./package.ktm.json")
if (ROOT / "boilerplate/runtime").exists() or list((ROOT / "boilerplate").rglob("node.package.json*")):
    raise SystemExit("boilerplate must not contain nested or duplicate package contracts")

manifest = load(ROOT / "package.ktm.json")
if (manifest["metadata"]["namespace"], manifest["metadata"]["name"]) != ("kinematic-trees", SDK_NAME):
    raise SystemExit(f"package.ktm.json: expected kinematic-trees/{SDK_NAME}")

print("Go SDK and template lock kinematic-trees/kt-messages@0.1.0 as data_types")

#!/usr/bin/env python3
"""Verify SDK and generated-project manifests lock one kt-messages release."""

import json
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
EXPECTED = {
    "owner": "kinematic-trees",
    "name": "kt-messages",
    "version": "0.1.0",
    "classification": "data_types",
}


def load(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8").replace("{{KTM_CREATE_RUN_ENVIRONMENTS_JSON}}", "[]"))


for relative in ("package.ktm.json", "boilerplate/package.ktm.json.template"):
    dependencies = load(ROOT / relative)["dependencies"]["packages"]
    matches = [item for item in dependencies if item.get("name") == "kt-messages"]
    if len(matches) != 1 or any(matches[0].get(key) != value for key, value in EXPECTED.items()):
        raise SystemExit(f"{relative}: expected one exact kt-messages data_types dependency")
    if not matches[0].get("environments"):
        raise SystemExit(f"{relative}: kt-messages environment mapping is empty")

print("Go SDK and template lock kinematic-trees/kt-messages@0.1.0 as data_types")

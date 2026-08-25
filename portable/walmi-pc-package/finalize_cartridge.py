from __future__ import annotations

import hashlib
import json
import sys
import zipfile
from pathlib import Path

BASE = "c6011afbb456b8fe1e6c7c6eda7857aec681cb8f"
DIST = Path("dist")
HOST = DIST / "host"
CARTRIDGE = DIST / "cartridge"
NATIVE = CARTRIDGE / "native/windows/WALMI_WINDOWS_FULL_HOST_v0_1.zip"


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def zip_tree(root: Path, out: Path) -> None:
    out.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED, compresslevel=6, allowZip64=True) as zf:
        for path in sorted(p for p in root.rglob("*") if p.is_file()):
            zf.write(path, path.relative_to(root).as_posix())
    with zipfile.ZipFile(out) as zf:
        bad = zf.testzip()
        if bad:
            raise SystemExit(f"bad zip member: {bad}")


if sys.argv[1:] == ["native"]:
    zip_tree(HOST, NATIVE)
    with zipfile.ZipFile(NATIVE) as zf:
        names = set(zf.namelist())
    required = {
        "bin/waldo.exe",
        "bin/waldo-axm-mirror.exe",
        "bin/waldo-mirror-review.exe",
        "runtime/python/python.exe",
        "START_WALMI_PC.cmd",
        "WALMI_LOCAL_SHELL.cmd",
        "models/README.txt",
        "checkpoints/README.txt",
        "experience/README.txt",
        "rollback/README.txt",
        "state/README.txt",
        "MODEL_WEIGHT_SCAN.json",
        "WINDOWS_RUNTIME_PATCH_RECEIPT.json",
    }
    missing = sorted(required - names)
    if missing:
        raise SystemExit(f"native host missing: {missing}")
    print("WINDOWS_FULL_HOST_PASS", NATIVE.stat().st_size, "bytes", len(names), "files")
    raise SystemExit(0)

if sys.argv[1:] != ["final"]:
    raise SystemExit("usage: finalize_cartridge.py native|final")

inventory = CARTRIDGE / "inventory"
audit = json.loads((inventory / "EXPERIMENT_STACK_AUDIT.json").read_text(encoding="utf-8"))
weights = json.loads((inventory / "MODEL_WEIGHT_SCAN.json").read_text(encoding="utf-8"))
unmodified = json.loads((inventory / "UNMODIFIED_VERIFICATION.json").read_text(encoding="utf-8"))

if not audit["allCriticalPresent"] or audit["mirrorWaldoBranchCount"] != 58:
    raise SystemExit("experiment stack audit is not complete")
if unmodified.get("commit") != BASE or unmodified.get("status") != "PASS" or unmodified.get("goTestAll") is not True:
    raise SystemExit("exact source verification receipt is not PASS")

manifest = {
    "schema": "walmi.cartridge/v1",
    "name": "WALMI-PC-FULL-STACK",
    "version": "0.1.0",
    "browserEntry": "runtime/browser-entry.js",
    "source": {
        "repository": "mike-axiom-mir/waldo-axm-mirror-research",
        "walmiBaseCommit": BASE,
        "currentStackArchive": "source/WALMI_CURRENT_STACK_c6011af.zip",
        "allExperimentHistory": "history/WALDO_ALL_EXPERIMENTS.bundle",
        "experimentBranchesAudited": audit["mirrorWaldoBranchCount"],
    },
    "stack": [
        "MIRROR",
        "WALDO",
        "HERMES",
        "EPHEMERAL_SPECIALISTS",
        "NEURAL_SOCKETS",
        "DETERMINIZATION_STEWARD",
        "WORKSHOP_CAPABILITY_INTAKE",
        "SERVICE_MODES",
        "GRAMMAR_GLASS",
        "CREATION_FABRIC",
        "PR67_OBJECT_ADAPTER",
        "PR68_RECORD_QUERY",
        "MODEL_LIFECYCLE",
        "TRAINING",
        "INFERENCE",
    ],
    "nativeHosts": [
        {
            "platform": "windows",
            "arch": "x64",
            "label": "WALMI Windows Full Host",
            "path": "native/windows/WALMI_WINDOWS_FULL_HOST_v0_1.zip",
            "fileName": "WALMI_WINDOWS_FULL_HOST_v0_1.zip",
            "sha256": sha256(NATIVE),
        }
    ],
    "runtime": {
        "go": {
            "target": "js/wasm",
            "wasm": "runtime/go/walmi-axm-mirror.wasm",
            "exec": "runtime/go/wasm_exec.js",
            "targetNeedsGoInstalled": False,
        },
        "windows": {
            "portablePython": True,
            "portablePyTorchCPU": True,
            "targetNeedsPythonInstalled": False,
            "trainingAndInferenceCodeBundled": True,
            "torchTitan": "SOURCE_RETAINED_WINDOWS_DISABLED_USE_PYTORCH",
        },
        "neural": {
            "startingWeightsBundled": weights["startingWeightsBundled"],
            "weightFiles": weights["recognizedWeightFiles"],
            "internetFallback": False,
        },
    },
    "experienceExport": True,
    "rollbackStorage": True,
    "offlineRequired": True,
    "authority": "NONE",
}
(CARTRIDGE / "walmi.manifest.json").write_text(json.dumps(manifest, indent=2), encoding="utf-8")
(CARTRIDGE / "README_FIRST.txt").write_text(
    "WALMI PC FULL STACK v0.1\n\n"
    "Open the single WALMI HTML launcher on PC, load this ZIP, choose FULL STRENGTH, "
    "extract the emitted Windows host ZIP, then run START_WALMI_PC.cmd.\n\n"
    "The exact current source and complete experiment Git history are included. "
    "Inventory receipts prove branch/file coverage and the starting-weight truth.\n",
    encoding="utf-8",
)

required = [
    "walmi.manifest.json",
    "runtime/browser-entry.js",
    "runtime/go/walmi-axm-mirror.wasm",
    "runtime/go/wasm_exec.js",
    "native/windows/WALMI_WINDOWS_FULL_HOST_v0_1.zip",
    "source/WALMI_CURRENT_STACK_c6011af.zip",
    "history/WALDO_ALL_EXPERIMENTS.bundle",
    "inventory/EXPERIMENT_STACK_AUDIT.json",
    "inventory/CURRENT_STACK_FILES.txt",
    "inventory/MIRROR_WALDO_BRANCHES.txt",
    "inventory/MODEL_WEIGHT_SCAN.json",
    "inventory/UNMODIFIED_VERIFICATION.json",
    "inventory/UNMODIFIED_GO_TESTS.txt",
    "inventory/GIT_BUNDLE_VERIFY.txt",
    "inventory/WINDOWS_RUNTIME_PATCH_RECEIPT.json",
    "inventory/PYTORCH_RUNTIME_VERIFY.txt",
    "inventory/WALDO_WINDOWS_VERSION.txt",
    "inventory/WALDO_WINDOWS_MIRROR_HELP.txt",
]
missing = [name for name in required if not (CARTRIDGE / name).is_file()]
if missing:
    raise SystemExit(f"FINAL_COMPLETENESS_HOLD {missing}")

rows = []
for path in sorted(p for p in CARTRIDGE.rglob("*") if p.is_file()):
    rows.append({"path": path.relative_to(CARTRIDGE).as_posix(), "bytes": path.stat().st_size, "sha256": sha256(path)})
(inventory / "CARTRIDGE_FILE_MANIFEST.json").write_text(
    json.dumps({"schema": "walmi.cartridge-files/v1", "count": len(rows), "files": rows}, indent=2),
    encoding="utf-8",
)

out = DIST / "WALMI_PC_FULL_STACK_v0_1.zip"
zip_tree(CARTRIDGE, out)
digest = sha256(out)
(DIST / "WALMI_PC_FULL_STACK_v0_1.sha256.txt").write_text(f"{digest}  {out.name}\n", encoding="utf-8")
print(
    "TRIPLE_CHECK_PASS",
    audit["mirrorWaldoBranchCount"],
    "branches",
    audit["currentTreeFileCount"],
    "current-tree files",
    len(rows),
    "cartridge payload files",
    "starting weights",
    weights["count"],
)
print("FINAL_WALMI_PC_ZIP_PASS", out.stat().st_size, digest)

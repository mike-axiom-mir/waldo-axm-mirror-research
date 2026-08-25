from __future__ import annotations

import shutil
from pathlib import Path

host = Path("dist/host")
for name in ["models", "checkpoints", "experience", "rollback", "state"]:
    folder = host / name
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "README.txt").write_text(
        f"WALMI local {name} storage. This receipt keeps the directory present in ZIP exports.\n",
        encoding="utf-8",
    )

(host / "START_WALMI_PC.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%PATH%"
set "PYTHONHOME=%CD%\runtime\python"
set "WALMI_HOME=%CD%"
echo WALMI PC FULL STACK
echo Local Go/Python/PyTorch runtime loaded from this folder.
echo.
bin\waldo.exe advisor WALMI
if errorlevel 1 cmd /k "echo WALMI shell ready. Try bin\waldo.exe --help and bin\waldo.exe mirror --help"
''',
    encoding="ascii",
)
(host / "WALMI_LOCAL_SHELL.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%PATH%"
set "PYTHONHOME=%CD%\runtime\python"
set "WALMI_HOME=%CD%"
cmd /k "echo WALMI local shell ready. Try bin\waldo.exe mirror --help"
''',
    encoding="ascii",
)
(host / "README_FIRST.txt").write_text(
    "WALMI WINDOWS FULL HOST v0.1\n\n"
    "Run START_WALMI_PC.cmd.\n"
    "Bundled: WALDO, AXM Mirror, Mirror Review, portable Python 3.12, CPU PyTorch, "
    "model/training/inference code, and local growth folders.\n"
    "No Go or Python installation is required on the target PC.\n\n"
    "Exact unmodified c6011af source and all experiment history are preserved in the OUTER WALMI cartridge.\n"
    "TorchTitan remains in source/history but the Windows runtime uses PyTorch.\n"
    "MODEL_WEIGHT_SCAN.json states whether starting weights existed. No weights are fabricated.\n",
    encoding="utf-8",
)

for name in ["MODEL_WEIGHT_SCAN.json"]:
    shutil.copy2(Path("dist/import/inventory") / name, host / name)
shutil.copy2("dist/WINDOWS_RUNTIME_PATCH_RECEIPT.json", host / "WINDOWS_RUNTIME_PATCH_RECEIPT.json")
for name in ["LICENSE", "NOTICE"]:
    shutil.copy2(name, host / name)

print("WALMI host layout written")

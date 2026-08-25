from __future__ import annotations

import shutil
from pathlib import Path

host = Path("dist/host")
for name in ["models", "checkpoints", "experience", "rollback", "state", "workspace", "assets"]:
    folder = host / name
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "README.txt").write_text(
        f"WALMI local {name} storage. This directory is part of the portable host.\n",
        encoding="utf-8",
    )

(host / "START_WALMI_PC.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%CD%\runtime\node;%PATH%"
set "PYTHONHOME=%CD%\runtime\python"
set "WALMI_HOME=%CD%"
echo WALMI PC FULL STACK v0.2
echo Local WALDO + Mirror + Hermes research stack, weight-learning runtime, Workspace Hand and Asset Hands loaded from this folder.
echo Workshop is NOT required.
echo.
bin\waldo.exe advisor WALMI
if errorlevel 1 cmd /k "echo WALMI shell ready. Try bin\waldo.exe --help, bin\waldo.exe mirror --help, WALMI_WORKSPACE_HAND.cmd or WALMI_ASSET_HAND.cmd"
''',
    encoding="ascii",
)
(host / "WALMI_LOCAL_SHELL.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
set "PATH=%CD%\runtime\python;%CD%\runtime\python\Scripts;%CD%\runtime\node;%PATH%"
set "PYTHONHOME=%CD%\runtime\python"
set "WALMI_HOME=%CD%"
cmd /k "echo WALMI local shell ready. Workshop is optional, not a dependency."
''',
    encoding="ascii",
)
(host / "WALMI_WORKSPACE_HAND.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
if "%~2"=="" (
  echo Usage: WALMI_WORKSPACE_HAND.cmd ^<workspace-folder^> ^<request.json^> [response.json]
  exit /b 2
)
if "%~3"=="" (
  runtime\python\python.exe tools\walmi_workspace_hand.py "%~1" "%~2"
) else (
  runtime\python\python.exe tools\walmi_workspace_hand.py "%~1" "%~2" "%~3"
)
''',
    encoding="ascii",
)
(host / "WALMI_ASSET_HAND.cmd").write_text(
    r'''@echo off
setlocal
cd /d "%~dp0"
if "%~1"=="" (
  echo Usage: WALMI_ASSET_HAND.cmd ^<request.json^> [response.json]
  exit /b 2
)
if "%~2"=="" (
  runtime\node\node.exe tools\walmi_asset_hand_runner.js "%~1"
) else (
  runtime\node\node.exe tools\walmi_asset_hand_runner.js "%~1" "%~2"
)
''',
    encoding="ascii",
)
(host / "README_FIRST.txt").write_text(
    "WALMI WINDOWS FULL HOST v0.2\n\n"
    "Run START_WALMI_PC.cmd.\n\n"
    "The host is standalone: Workshop is not required. It includes WALDO, AXM Mirror, Mirror Review, "
    "portable Python/PyTorch, a portable Node runtime, the WALMI Workspace Hand, the existing WALMI inner-asset path, "
    "and a pinned local copy of AXM Asset Fabric/Asset Hands capability code.\n\n"
    "Workspace access is opt-in per folder. If no folder is attached WALMI still boots normally. "
    "Writes are candidate-only, hash-bound, transactional, symlink-refusing, .git-refusing and rollback-on-error.\n\n"
    "Asset generation is local. Native Blender/Godot/Unity/Unreal/FreeCAD/FFmpeg-style bridges remain optional and only work "
    "when the target program/tool exists and the hand's own permission checks pass.\n\n"
    "MODEL_WEIGHT_SCAN.json states whether starting weights existed. No weights are fabricated.\n",
    encoding="utf-8",
)

for name in ["MODEL_WEIGHT_SCAN.json", "ASSET_CAPABILITY_RECEIPT.json", "WORKSPACE_HAND_SMOKE.json"]:
    source = Path("dist/inventory") / name
    if source.is_file():
        shutil.copy2(source, host / name)
if Path("dist/WINDOWS_RUNTIME_PATCH_RECEIPT.json").is_file():
    shutil.copy2("dist/WINDOWS_RUNTIME_PATCH_RECEIPT.json", host / "WINDOWS_RUNTIME_PATCH_RECEIPT.json")
for name in ["LICENSE", "NOTICE"]:
    shutil.copy2(name, host / name)

print("WALMI v0.2 host layout written")

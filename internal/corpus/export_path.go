// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package corpus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SafeExportFilePath resolves a normalized export-relative file path and
// rejects symlinks or non-directory parents beneath root. When createParents
// is true, missing parent directories are created one component at a time so
// an existing symlink is never accepted as an export directory.
func SafeExportFilePath(root, relative string, createParents bool) (string, error) {
	converted := filepath.FromSlash(relative)
	clean := filepath.Clean(converted)
	if relative == "" || strings.Contains(relative, "\\") || filepath.IsAbs(converted) || filepath.ToSlash(clean) != relative || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid export path %q", relative)
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootInfo, err := os.Stat(absoluteRoot)
	if os.IsNotExist(err) && createParents {
		if err := os.MkdirAll(absoluteRoot, 0o755); err != nil {
			return "", err
		}
		rootInfo, err = os.Stat(absoluteRoot)
	}
	if err != nil {
		return "", err
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf("export root %s is not a directory", absoluteRoot)
	}

	current := absoluteRoot
	parent := filepath.Dir(clean)
	if parent != "." {
		for _, component := range strings.Split(parent, string(filepath.Separator)) {
			current = filepath.Join(current, component)
			info, statErr := os.Lstat(current)
			if os.IsNotExist(statErr) && createParents {
				if mkdirErr := os.Mkdir(current, 0o755); mkdirErr != nil && !os.IsExist(mkdirErr) {
					return "", mkdirErr
				}
				info, statErr = os.Lstat(current)
			}
			if statErr != nil {
				return "", statErr
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("export path %s has symlink parent %s", relative, current)
			}
			if !info.IsDir() {
				return "", fmt.Errorf("export path %s has non-directory parent %s", relative, current)
			}
		}
	}

	path := filepath.Join(absoluteRoot, clean)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return path, nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("export file %s is a symlink", relative)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("export file %s is not a regular file", relative)
	}
	return path, nil
}

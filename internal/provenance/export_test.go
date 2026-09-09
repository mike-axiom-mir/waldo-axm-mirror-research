// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openwaldo/waldo/internal/corpus"
	"github.com/openwaldo/waldo/internal/index"
)

func TestWriteCorpusExport(t *testing.T) {
	destination := t.TempDir()
	document := exportFixture("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 4)
	if err := WriteCorpusExport(destination, document); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "EXPORT.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded CorpusExport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Kind != "waldo-corpus-export" || decoded.Format != "native" || decoded.Generated != "2026-08-04T12:00:00Z" || len(decoded.Files) != 1 {
		t.Fatalf("decoded export = %+v", decoded)
	}
}

func TestVerifyCorpusExportHashesFiles(t *testing.T) {
	destination := t.TempDir()
	content := []byte("native object")
	digestArray := sha256.Sum256(content)
	digest := hex.EncodeToString(digestArray[:])
	document := exportFixture(digest, int64(len(content)))
	filePath := filepath.Join(destination, filepath.FromSlash(document.Files[0].Path))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteCorpusExport(destination, document); err != nil {
		t.Fatal(err)
	}
	_, report, err := VerifyCorpusExport(destination)
	if err != nil {
		t.Fatal(err)
	}
	if report.Files != 1 || report.Bytes != int64(len(content)) {
		t.Fatalf("verification = %+v", report)
	}
	if err := os.WriteFile(filePath, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := VerifyCorpusExport(destination); err == nil {
		t.Fatal("expected corrupt export failure")
	}
}

func TestVerifyCorpusExportRejectsSymlinkedData(t *testing.T) {
	content := []byte("native object")
	digestArray := sha256.Sum256(content)
	digest := hex.EncodeToString(digestArray[:])
	document := exportFixture(digest, int64(len(content)))

	for _, test := range []struct {
		name string
		link func(t *testing.T, destination, outside, relative string)
	}{
		{
			name: "file",
			link: func(t *testing.T, destination, outside, relative string) {
				t.Helper()
				external := filepath.Join(outside, "object.parquet")
				if err := os.WriteFile(external, content, 0o644); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(destination, filepath.FromSlash(relative))
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(external, path); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			},
		},
		{
			name: "parent directory",
			link: func(t *testing.T, destination, outside, relative string) {
				t.Helper()
				external := filepath.Join(outside, filepath.FromSlash(strings.TrimPrefix(relative, "data/")))
				if err := os.MkdirAll(filepath.Dir(external), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(external, content, 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(destination, "data")); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			destination := t.TempDir()
			if err := WriteCorpusExport(destination, document); err != nil {
				t.Fatal(err)
			}
			test.link(t, destination, t.TempDir(), document.Files[0].Path)
			if _, _, err := VerifyCorpusExport(destination); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("VerifyCorpusExport() error = %v, want symlink rejection", err)
			}
		})
	}
}

func TestLoadCorpusExportRejectsSymlinkedDocument(t *testing.T) {
	external := t.TempDir()
	document := exportFixture("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 4)
	if err := WriteCorpusExport(external, document); err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	if err := os.Symlink(filepath.Join(external, "EXPORT.json"), filepath.Join(destination, "EXPORT.json")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, _, err := LoadCorpusExport(destination); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("LoadCorpusExport() error = %v, want symlink rejection", err)
	}
}

func TestCorpusExportValidationRejectsTraversalAndIncorrectTotals(t *testing.T) {
	document := exportFixture("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 4)
	document.Files[0].Path = "../escape.parquet"
	if err := document.Validate(); err == nil {
		t.Fatal("expected traversal path failure")
	}
	document = exportFixture("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 4)
	document.BOM.Totals.Tokens++
	if err := document.Validate(); err == nil {
		t.Fatal("expected totals failure")
	}
}

func TestCorpusExportSchemaOneGolden(t *testing.T) {
	document := exportFixture("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 4)
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(append(data, '\n'))
	got := hex.EncodeToString(digest[:])
	const want = "5b15d0031efcb93086b060b3f5b2bd28c1b4d5a48aa7794ab860d0c448a8066b"
	if got != want {
		t.Fatalf("schema-1 golden hash = %s, want %s", got, want)
	}
}

func exportFixture(objectHash string, objectBytes int64) CorpusExport {
	measure := index.Measures{Shards: 1, Docs: 1, Tokens: 2, Bytes: objectBytes}
	conversion := index.Conversion{Tool: "fixture", Version: "1", Profile: "text", Recipe: "fixture/v1", Tokenizer: "byte"}
	bom := corpus.BOM{
		Kind: "openwaldo-bom", Schema: 1, Subject: "corpus", Paths: []string{"books"},
		Manifests: []corpus.ManifestPin{{
			Path: "books/books.json", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Name: "books", Title: "Books", Description: "Fixture.", License: "CC0-1.0",
			Format: "parquet", RecordSchema: 1, ConvertedBy: conversion,
			Sources: []index.Source{{Name: "source", Source: "Fixture", URL: "https://example.test", SHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}},
			Totals:  measure, Licenses: map[string]index.Measures{"CC0-1.0": measure},
		}},
		Shards: []corpus.ShardPin{{
			Manifest: "books/books.json", URL: "https://objects.test/item", SHA256: objectHash,
			Format: "parquet", RecordSchema: 1, License: "CC0-1.0", Sources: []string{"source"},
			ConvertedBy: conversion, Docs: 1, Tokens: 2, Bytes: objectBytes,
		}},
		Totals: measure, Licenses: map[string]index.Measures{"CC0-1.0": measure},
	}
	files := []corpus.ExportedFile{{
		Path: "data/books/item.parquet", Manifest: "books/books.json", ObjectSHA256: objectHash,
		SHA256: objectHash, Format: "parquet", License: "CC0-1.0", Docs: 1, Tokens: 2,
		ObjectBytes: objectBytes, Bytes: objectBytes,
	}}
	return NewCorpusExport(bom, "native", files, time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC))
}

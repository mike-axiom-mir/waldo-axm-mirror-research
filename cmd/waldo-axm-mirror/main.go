package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

func main() {
	if len(os.Args) != 3 && len(os.Args) != 4 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "seal":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = sealFile(os.Args[2], os.Args[3])
	case "verify":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		err = verifyFile(os.Args[2])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror verify <sealed.json>")
}

func sealFile(inputPath, outputPath string) error {
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.Seal(draft)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func verifyFile(path string) error {
	var sealed axmmirror.SealedBehaviorEvidence
	if err := readStrictJSON(path, &sealed); err != nil {
		return err
	}
	if err := sealed.Verify(); err != nil {
		return err
	}
	fmt.Println("OK", sealed.SHA256)
	return nil
}

func readStrictJSON(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if decoder.More() {
		return fmt.Errorf("decode %s: trailing JSON content", path)
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

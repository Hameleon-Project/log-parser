package service

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func readLogBytes(path string) ([]byte, error) {
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		return readFirstDiagFileFromZip(path)
	}
	return os.ReadFile(path)
}

func readFirstDiagFileFromZip(zipPath string) ([]byte, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer func() { _ = zr.Close() }()

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		lower := strings.ToLower(filepath.Base(f.Name))
		if strings.Contains(lower, "db_csv") || strings.HasSuffix(lower, ".db_csv") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			b, readErr := io.ReadAll(rc)
			closeErr := rc.Close()
			if readErr != nil {
				return nil, readErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			return b, nil
		}
	}
	return nil, fmt.Errorf("archive does not contain an ibdiagnet *db_csv* export")
}

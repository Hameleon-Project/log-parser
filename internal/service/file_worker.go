package service

import (
	"os"
	"path/filepath"
)

const DataDir = "data"

func (s *ParserService) ScanDataDir() ([]string, error) {
	files, err := os.ReadDir(DataDir)
	if err != nil {
		return nil, err
	}

	var filePaths []string
	for _, file := range files {
		if !file.IsDir() {
			filePaths = append(filePaths, filepath.Join(DataDir, file.Name()))
		}
	}
	return filePaths, nil
}

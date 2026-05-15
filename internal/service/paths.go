package service

import (
	"errors"
	"path/filepath"
	"strings"
)

var ErrInvalidDataPath = errors.New("path must be relative to data/ without '..' segments")

func ResolveUnderDataDir(baseDir, userPath string) (string, error) {
	clean := filepath.Clean(userPath)
	slash := filepath.ToSlash(clean)
	if strings.Contains(slash, "..") {
		return "", ErrInvalidDataPath
	}
	if slash != "data" && !strings.HasPrefix(slash, "data/") {
		return "", ErrInvalidDataPath
	}
	full := filepath.Join(baseDir, filepath.FromSlash(slash))

	absData, err := filepath.Abs(filepath.Join(baseDir, "data"))
	if err != nil {
		return "", err
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absData, absFull)
	if err != nil {
		return "", ErrInvalidDataPath
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrInvalidDataPath
	}
	return full, nil
}

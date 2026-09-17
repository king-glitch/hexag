package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed template.tar.gz
var templateArchive []byte

type templateFile struct {
	Path    string
	IsDir   bool
	Mode    os.FileMode
	Content []byte
}

func loadTemplateFiles() ([]templateFile, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(templateArchive))
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded template archive: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var files []templateFile

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		cleanPath := filepath.Clean(header.Name)
		if cleanPath == "." || cleanPath == "/" {
			continue
		}
		cleanPath = strings.TrimPrefix(cleanPath, "/")

		if header.Typeflag == tar.TypeDir {
			files = append(files, templateFile{
				Path:  cleanPath,
				IsDir: true,
				Mode:  header.FileInfo().Mode(),
			})
			continue
		}

		content, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("failed to read file content %s: %w", header.Name, err)
		}

		files = append(files, templateFile{
			Path:    cleanPath,
			IsDir:   false,
			Mode:    header.FileInfo().Mode(),
			Content: content,
		})
	}

	return files, nil
}

func getTemplateFile(targetPath string) (*templateFile, error) {
	files, err := loadTemplateFiles()
	if err != nil {
		return nil, err
	}
	cleanTarget := filepath.Clean(targetPath)
	for _, f := range files {
		if filepath.Clean(f.Path) == cleanTarget {
			return &f, nil
		}
	}
	return nil, fmt.Errorf("template file not found: %s", targetPath)
}

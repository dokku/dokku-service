package template

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ExtractTemplate extracts a service template to the specified path
func ExtractTemplate(entry ServiceTemplate, extractPath string) error {
	templatePath := entry.Name
	entries, err := ReadDir(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, dirEntry := range entries {
		if dirEntry.IsDir() {
			err := WriteDir(entry, dirEntry, "", extractPath)
			if err != nil {
				return fmt.Errorf("failed to write directory: %w", err)
			}

			continue
		}

		writePath := filepath.Join(extractPath, dirEntry.Name())
		err := WriteFile(entry, dirEntry, "", writePath)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

func WriteDir(entry ServiceTemplate, dirEntry fs.DirEntry, basePath string, extractPath string) error {
	if !dirEntry.IsDir() {
		return nil
	}

	newBasePath := filepath.Join(basePath, dirEntry.Name())
	err := os.MkdirAll(filepath.Join(extractPath, newBasePath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	entries, err := ReadDir(filepath.Join(entry.Name, newBasePath))
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, dirEntry := range entries {
		if dirEntry.IsDir() {
			err := WriteDir(entry, dirEntry, newBasePath, extractPath)
			if err != nil {
				return fmt.Errorf("failed to write directory: %w", err)
			}

			continue
		}

		writePath := filepath.Join(extractPath, newBasePath, dirEntry.Name())
		err := WriteFile(entry, dirEntry, newBasePath, writePath)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

func WriteFile(entry ServiceTemplate, dirEntry fs.DirEntry, basePath string, writePath string) error {
	if dirEntry.IsDir() {
		return fmt.Errorf("cannot write directory: %s", dirEntry.Name())
	}

	contents, err := ReadFile(filepath.Join(entry.Name, basePath, dirEntry.Name()))
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	handle, err := os.Create(writePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	_, err = handle.Write(contents)
	if err != nil {
		return fmt.Errorf("failed to write content to file: %w", err)
	}

	// todo: set permissions

	return nil
}

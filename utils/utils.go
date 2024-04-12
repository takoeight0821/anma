package utils

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/takoeight0821/anma/token"
)

// PosError represents an error that occurred at a specific position in the code.
type PosError struct {
	Where token.Token // The token indicating the position of the error.
	Err   error       // The underlying error.
}

func (e PosError) Error() string {
	if e.Where.Kind == token.EOF {
		return "at end: " + e.Err.Error()
	}

	return fmt.Sprintf("at %v: `%s`\n\t%s", e.Where.Location, e.Where.Lexeme, e.Err.Error())
}

func (e PosError) Unwrap() error {
	return e.Err
}

func FindSourceFiles(path string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(path, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".anma" {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find source files: %w", err)
	}

	return files, nil
}

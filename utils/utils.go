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
		return fmt.Sprintf("at end: %s", e.Err.Error())
	}

	return fmt.Sprintf("at %d: `%s`, %s", e.Where.Line, e.Where.Lexeme, e.Err.Error())
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

	return files, err
}

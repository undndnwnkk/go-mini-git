package service

import (
	"errors"
	"fmt"
	"os"
)

func ValidateRoot(root string) error {
	if root == "" {
		return ErrEmptyPath
	}

	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("validate root: %w", ErrNotExist)
		}

		return err
	}

	if !info.IsDir() {
		return ErrNotDir
	}

	return nil

}

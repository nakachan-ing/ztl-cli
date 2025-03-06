package util

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func OpenEditor(filePath string, config model.Config) error {
	c := exec.Command(config.Editor, filePath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to open editor (%s): %w", filePath, err)
	}
	return nil
}

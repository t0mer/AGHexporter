package instances

import (
	"fmt"
	"os"
	"strings"
)

// readSecret reads a secret from a file, trims surrounding whitespace, and
// never logs the file contents. The resolved path is only logged at DEBUG level.
func readSecret(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading secret file %q: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

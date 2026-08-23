package core

import (
	"os"
)

// stat is os.Stat aliased for test shims.
func stat(path string) (os.FileInfo, error) { return os.Stat(path) }

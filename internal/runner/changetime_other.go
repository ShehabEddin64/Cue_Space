//go:build !darwin

package runner

import (
	"io/fs"
	"time"
)

func changeTime(fs.FileInfo) time.Time { return time.Time{} }

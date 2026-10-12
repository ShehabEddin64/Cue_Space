package runner

import (
	"io/fs"
	"syscall"
	"time"
)

func changeTime(info fs.FileInfo) time.Time {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Ctimespec.Unix())
	}
	return time.Time{}
}

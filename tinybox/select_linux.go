//go:build linux

package tb

import "syscall"

func selectRead(fd int, set *syscall.FdSet, tv *syscall.Timeval) (int, error) {
	return syscall.Select(fd+1, set, nil, nil, tv)
}

//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package tb

import "syscall"

const (
	TCGETS     = syscall.TIOCGETA
	TCSETS     = syscall.TIOCSETA
	TIOCGWINSZ = syscall.TIOCGWINSZ
	ICANON     = syscall.ICANON
	ECHO       = syscall.ECHO
	ISIG       = syscall.ISIG
	IEXTEN     = syscall.IEXTEN
	BRKINT     = syscall.BRKINT
	ICRNL      = syscall.ICRNL
	INPCK      = syscall.INPCK
	ISTRIP     = syscall.ISTRIP
	IXON       = syscall.IXON
	OPOST      = syscall.OPOST
	CS8        = syscall.CS8
	VMIN       = syscall.VMIN
	VTIME      = syscall.VTIME
	F_GETFL    = syscall.F_GETFL
	F_SETFL    = syscall.F_SETFL
	O_NONBLOCK = syscall.O_NONBLOCK
)

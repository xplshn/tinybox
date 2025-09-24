/* MIT License

Copyright (c) 2025 Sebastian <sebastian.michalk@pm.me>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE. */

/* tinybox */

package tb

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"
)

const (
	TCGETS     = 0x5401
	TCSETS     = 0x5402
	TIOCGWINSZ = 0x5413
	ICANON     = 0x2
	ECHO       = 0x8
	ISIG       = 0x1
	IEXTEN     = 0x8000
	BRKINT     = 0x2
	ICRNL      = 0x100
	INPCK      = 0x10
	ISTRIP     = 0x20
	IXON       = 0x400
	OPOST      = 0x1
	CS8        = 0x30
	VMIN       = 6
	VTIME      = 5
	F_GETFL    = 3
	F_SETFL    = 4
	O_NONBLOCK = 0x800

	ESC = "\033"
	BEL = "\x07"

	ClearScreen     = ESC + "[2J"
	ClearToEOL      = ESC + "[K"
	MoveCursor      = ESC + "[%d;%dH"
	SaveCursor      = ESC + "[s"
	RestoreCursor   = ESC + "[u"
	HideCursor      = ESC + "[?25l"
	ShowCursor      = ESC + "[?25h"
	AlternateScreen = ESC + "[?1049h"
	NormalScreen    = ESC + "[?1049l"
	QueryCursorPos  = ESC + "[6n"

	EnableMouseMode     = ESC + "[?1000h" + ESC + "[?1002h" + ESC + "[?1015h" + ESC + "[?1006h"
	DisableMouseMode    = ESC + "[?1000l" + ESC + "[?1002l" + ESC + "[?1015l" + ESC + "[?1006l"
	EnableBracketPaste  = ESC + "[?2004h"
	DisableBracketPaste = ESC + "[?2004l"

	ResetColor = ESC + "[0m"
	SetFgColor = ESC + "[38;5;%dm"
	SetBgColor = ESC + "[48;5;%dm"

	SetBold        = ESC + "[1m"
	SetItalic      = ESC + "[3m"
	SetUnderline   = ESC + "[4m"
	SetReverse     = ESC + "[7m"
	UnsetBold      = ESC + "[22m"
	UnsetItalic    = ESC + "[23m"
	UnsetUnderline = ESC + "[24m"
	UnsetReverse   = ESC + "[27m"

	BoxTopLeft     = '┌'
	BoxTopRight    = '┐'
	BoxBottomLeft  = '└'
	BoxBottomRight = '┘'
	BoxHorizontal  = '─'
	BoxVertical    = '│'

	CursorBlock     = 1
	CursorLine      = 3
	CursorUnderline = 5
)

var (
	seqSetBold        = []byte(SetBold)
	seqUnsetBold      = []byte(UnsetBold)
	seqSetItalic      = []byte(SetItalic)
	seqUnsetItalic    = []byte(UnsetItalic)
	seqSetUnderline   = []byte(SetUnderline)
	seqUnsetUnderline = []byte(UnsetUnderline)
	seqSetReverse     = []byte(SetReverse)
	seqUnsetReverse   = []byte(UnsetReverse)
	resetColorSeq     = []byte(ResetColor)
)

type termios struct {
	Iflag, Oflag, Cflag, Lflag uint32
	Line                       uint8
	Cc                         [32]uint8
	Ispeed, Ospeed             uint32
}

type winsize struct {
	Row, Col, Xpixel, Ypixel uint16
}

type Cell struct {
	Ch     rune
	Fg     int
	Bg     int
	Bold   bool
	Italic bool
	Under  bool
	Rev    bool
	Dirty  bool
}

type Buffer struct {
	Width  int
	Height int
	Cells  [][]Cell
}

type Event struct {
	Type   EventType
	Key    Key
	Ch     rune
	X      int
	Y      int
	Button MouseButton
	Mod    KeyMod
	Press  bool
}

type EventType int

const (
	EventKey EventType = iota
	EventMouse
	EventResize
	EventPaste
)

type Key int

const (
	KeyCtrlC Key = iota + 1
	KeyCtrlD
	KeyEscape
	KeyEnter
	KeyTab
	KeyBackspace
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyCtrlA
	KeyCtrlE
	KeyCtrlK
	KeyCtrlU
	KeyCtrlW
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyDelete
)

type KeyMod int

const (
	ModShift KeyMod = 1 << iota
	ModAlt
	ModCtrl
)

type MouseButton int

const (
	MouseLeft MouseButton = iota
	MouseMiddle
	MouseRight
	MouseWheelUp
	MouseWheelDown
)

type Terminal struct {
	origTermios   termios
	buffer        Buffer
	backBuffer    Buffer
	savedBuffer   [][]Cell
	width         int
	height        int
	initialized   bool
	isRaw         bool
	mouseEnabled  bool
	pasteEnabled  bool
	eventQueue    []Event
	currentFg     int
	currentBg     int
	currentBold   bool
	currentItalic bool
	currentUnder  bool
	currentRev    bool
	cursorX       int
	cursorY       int
	cursorVisible bool
	cursorStyle   int
	escDelay      int
	sigwinchCh    chan os.Signal
	sigcontCh     chan os.Signal
}

var term Terminal

func getTermios(fd int) (*termios, error) {
	var t termios
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), TCGETS, uintptr(unsafe.Pointer(&t)))
	if e != 0 {
		return nil, e
	}
	return &t, nil
}

func setTermios(fd int, t *termios) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), TCSETS, uintptr(unsafe.Pointer(t)))
	if e != 0 {
		return e
	}
	return nil
}

func enableRawMode() error {
	orig, err := getTermios(int(syscall.Stdin))
	if err != nil {
		return err
	}
	term.origTermios = *orig
	raw := *orig
	raw.Lflag &= ^uint32(ECHO | ICANON | ISIG | IEXTEN)
	raw.Iflag &= ^uint32(BRKINT | ICRNL | INPCK | ISTRIP | IXON)
	raw.Oflag &= ^uint32(OPOST)
	raw.Cflag |= CS8
	raw.Cc[VMIN] = 1
	raw.Cc[VTIME] = 0
	return setTermios(int(syscall.Stdin), &raw)
}

func disableRawMode() error {
	return setTermios(int(syscall.Stdin), &term.origTermios)
}

func queryTermSize() (int, int, error) {
	writeString("\033[999;999H\033[6n")

	var buf [32]byte
	fd := int(syscall.Stdin)

	fdSet := &syscall.FdSet{}
	fdSet.Bits[fd/64] |= 1 << (uint(fd) % 64)
	tv := syscall.Timeval{Sec: 1, Usec: 0}

	n, err := syscall.Select(fd+1, fdSet, nil, nil, &tv)
	if err != nil || n == 0 {
		return 80, 24, fmt.Errorf("terminal size query timeout")
	}

	n, err = syscall.Read(syscall.Stdin, buf[:])
	if err != nil || n < 6 {
		return 80, 24, fmt.Errorf("failed to read terminal response")
	}

	response := string(buf[:n])
	if len(response) >= 6 && response[0] == '\x1b' && response[1] == '[' {
		var row, col int
		if _, err := fmt.Sscanf(response[2:], "%d;%dR", &row, &col); err == nil {
			return col, row, nil
		}
	}
	return 80, 24, nil
}

func getTermSize() (int, int, error) {
	cols, _ := strconv.Atoi(os.Getenv("COLUMNS"))
	lines, _ := strconv.Atoi(os.Getenv("LINES"))
	if cols > 0 && lines > 0 {
		return cols, lines, nil
	}
	var ws winsize
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdout), TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if e == 0 {
		return int(ws.Col), int(ws.Row), nil
	}
	return queryTermSize()
}

func handleSigwinch() {
	for range term.sigwinchCh {
		width, height, err := getTermSize()
		if err == nil && (width != term.width || height != term.height) {
			term.width = width
			term.height = height
			term.buffer = initBuffer(width, height)
			term.backBuffer = initBuffer(width, height)

			if len(term.eventQueue) < cap(term.eventQueue) {
				term.eventQueue = append(term.eventQueue, Event{Type: EventResize})
			}
		}
	}
}

func handleSigcont() {
	for range term.sigcontCh {
		Resume()
	}
}

func writeString(s string) {
	syscall.Write(syscall.Stdout, []byte(s))
}

func initBuffer(width, height int) Buffer {
	cells := make([][]Cell, height)
	for i := range cells {
		cells[i] = make([]Cell, width)
		for j := range cells[i] {
			cells[i][j] = Cell{Ch: ' ', Fg: 7, Bg: 0, Dirty: true}
		}
	}
	return Buffer{Width: width, Height: height, Cells: cells}
}

func Init() error {
	if term.initialized {
		return fmt.Errorf("terminal already initialized")
	}

	width, height, err := getTermSize()
	if err != nil {
		return err
	}

	err = enableRawMode()
	if err != nil {
		return err
	}

	term.width = width
	term.height = height
	term.buffer = initBuffer(width, height)
	term.backBuffer = initBuffer(width, height)
	term.eventQueue = make([]Event, 0, 256)
	term.initialized = true
	term.isRaw = true
	term.currentFg = 7
	term.currentBg = 0
	term.cursorVisible = true
	term.cursorStyle = CursorBlock
	term.escDelay = 25

	term.sigwinchCh = make(chan os.Signal, 1)
	term.sigcontCh = make(chan os.Signal, 1)
	signal.Notify(term.sigwinchCh, syscall.SIGWINCH)
	signal.Notify(term.sigcontCh, syscall.SIGCONT)
	go handleSigwinch()
	go handleSigcont()

	writeString(AlternateScreen)
	writeString(HideCursor)
	writeString(ClearScreen)

	return nil
}

func Close() error {
	if !term.initialized {
		return nil
	}

	if term.mouseEnabled {
		writeString(DisableMouseMode)
	}
	if term.pasteEnabled {
		writeString(DisableBracketPaste)
	}

	signal.Stop(term.sigwinchCh)
	signal.Stop(term.sigcontCh)
	close(term.sigwinchCh)
	close(term.sigcontCh)

	writeString(ShowCursor)
	writeString(NormalScreen)
	writeString(ResetColor)

	err := disableRawMode()
	term.initialized = false
	term.isRaw = false
	return err
}

func Clear() {
	term.currentFg = 7
	term.currentBg = 0
	term.currentBold = false
	term.currentItalic = false
	term.currentUnder = false
	term.currentRev = false

	for y := 0; y < term.height; y++ {
		for x := 0; x < term.width; x++ {
			term.buffer.Cells[y][x] = Cell{Ch: ' ', Fg: 7, Bg: 0, Dirty: true}
			term.backBuffer.Cells[y][x] = Cell{Ch: 'X', Fg: 0, Bg: 0, Dirty: false}
		}
	}
}

func SetCell(x, y int, ch rune, fg, bg int) {
	if x < 0 || x >= term.width || y < 0 || y >= term.height {
		return
	}
	cell := &term.buffer.Cells[y][x]
	if cell.Ch != ch || cell.Fg != fg || cell.Bg != bg ||
		cell.Bold != term.currentBold || cell.Italic != term.currentItalic ||
		cell.Under != term.currentUnder || cell.Rev != term.currentRev {
		cell.Ch = ch
		cell.Fg = fg
		cell.Bg = bg
		cell.Bold = term.currentBold
		cell.Italic = term.currentItalic
		cell.Under = term.currentUnder
		cell.Rev = term.currentRev
		cell.Dirty = true
	}
}

func Present() {
	if term.width == 0 || term.height == 0 {
		return
	}

	output := make([]byte, 0, term.width*term.height)
	lastY, lastX := -1, -1
	activeFg, activeBg := -1, -1
	activeBold, activeItalic, activeUnder, activeRev := false, false, false, false
	var runeBuf [utf8.UTFMax]byte
	dirtyWritten := false

	for y := 0; y < term.height; y++ {
		for x := 0; x < term.width; x++ {
			curr := &term.buffer.Cells[y][x]
			back := &term.backBuffer.Cells[y][x]

			if !curr.Dirty {
				continue
			}

			if curr.Ch == back.Ch && curr.Fg == back.Fg && curr.Bg == back.Bg &&
				curr.Bold == back.Bold && curr.Italic == back.Italic &&
				curr.Under == back.Under && curr.Rev == back.Rev {
				curr.Dirty = false
				continue
			}

			if lastY != y || lastX != x {
				output = appendCursorMove(output, y+1, x+1)
			}

			if curr.Bold != activeBold {
				if curr.Bold {
					output = append(output, seqSetBold...)
				} else {
					output = append(output, seqUnsetBold...)
				}
				activeBold = curr.Bold
			}
			if curr.Italic != activeItalic {
				if curr.Italic {
					output = append(output, seqSetItalic...)
				} else {
					output = append(output, seqUnsetItalic...)
				}
				activeItalic = curr.Italic
			}
			if curr.Under != activeUnder {
				if curr.Under {
					output = append(output, seqSetUnderline...)
				} else {
					output = append(output, seqUnsetUnderline...)
				}
				activeUnder = curr.Under
			}
			if curr.Rev != activeRev {
				if curr.Rev {
					output = append(output, seqSetReverse...)
				} else {
					output = append(output, seqUnsetReverse...)
				}
				activeRev = curr.Rev
			}

			if curr.Fg != activeFg {
				output = appendSet256Color(output, true, curr.Fg)
				activeFg = curr.Fg
			}
			if curr.Bg != activeBg {
				output = appendSet256Color(output, false, curr.Bg)
				activeBg = curr.Bg
			}

			n := utf8.EncodeRune(runeBuf[:], curr.Ch)
			output = append(output, runeBuf[:n]...)

			*back = *curr
			curr.Dirty = false
			dirtyWritten = true
			lastY, lastX = y, x+1
		}
	}

	if dirtyWritten {
		output = append(output, resetColorSeq...)
		activeFg, activeBg = 7, 0
		activeBold, activeItalic, activeUnder, activeRev = false, false, false, false
	}

	if term.cursorVisible && (term.cursorX >= 0 && term.cursorY >= 0) {
		output = appendCursorMove(output, term.cursorY+1, term.cursorX+1)
	}

	if len(output) > 0 {
		syscall.Write(syscall.Stdout, output)
	}
}

func appendCursorMove(out []byte, row, col int) []byte {
	if row < 1 {
		row = 1
	}
	if col < 1 {
		col = 1
	}
	out = append(out, '', '[')
	out = appendInt(out, row)
	out = append(out, ';')
	out = appendInt(out, col)
	return append(out, 'H')
}

func appendSet256Color(out []byte, fg bool, value int) []byte {
	if value < 0 {
		value = 0
	} else if value > 255 {
		value = 255
	}
	out = append(out, '', '[')
	if fg {
		out = append(out, '3', '8')
	} else {
		out = append(out, '4', '8')
	}
	out = append(out, ';', '5', ';')
	out = appendInt(out, value)
	return append(out, 'm')
}

func appendInt(out []byte, value int) []byte {
	return strconv.AppendInt(out, int64(value), 10)
}

func DrawTextLeft(y int, text string, fg, bg int) {
	for i, ch := range text {
		if i < term.width {
			SetCell(i, y, ch, fg, bg)
		}
	}
}

func DrawTextCenter(y int, text string, fg, bg int) {
	startX := (term.width - len(text)) / 2
	if startX < 0 {
		startX = 0
	}
	for i, ch := range text {
		x := startX + i
		if x < term.width {
			SetCell(x, y, ch, fg, bg)
		}
	}
}

func DrawTextRight(y int, text string, fg, bg int) {
	startX := term.width - len(text)
	if startX < 0 {
		startX = 0
	}
	for i, ch := range text {
		x := startX + i
		if x < term.width && x >= 0 {
			SetCell(x, y, ch, fg, bg)
		}
	}
}

func ClearLine(y int) {
	for x := 0; x < term.width; x++ {
		SetCell(x, y, ' ', 7, 0)
	}
}

func GetTerminalSize() (width, height int) {
	return term.width, term.height
}

func PollEvent() (Event, error) {
	if len(term.eventQueue) > 0 {
		evt := term.eventQueue[0]
		term.eventQueue = term.eventQueue[1:]
		return evt, nil
	}

	var buf [16]byte
	n, err := syscall.Read(syscall.Stdin, buf[:])
	if err != nil {
		return Event{}, err
	}
	if n == 0 {
		return Event{}, fmt.Errorf("no input")
	}

	return parseInput(buf[:n])
}

func PollEventTimeout(timeout time.Duration) (Event, error) {
	if len(term.eventQueue) > 0 {
		evt := term.eventQueue[0]
		term.eventQueue = term.eventQueue[1:]
		return evt, nil
	}

	fd := int(syscall.Stdin)
	fdSet := &syscall.FdSet{}
	fdSet.Bits[fd/64] |= 1 << (uint(fd) % 64)

	tv := syscall.Timeval{
		Sec:  int64(timeout / time.Second),
		Usec: int64((timeout % time.Second) / time.Microsecond),
	}

	n, err := syscall.Select(fd+1, fdSet, nil, nil, &tv)
	if err != nil {
		return Event{}, err
	}
	if n == 0 {
		return Event{}, fmt.Errorf("timeout")
	}

	return PollEvent()
}

func parseSGRMouse(buf []byte) (Event, error) {
	// SGR format: \033[<button;x;y[Mm]
	if len(buf) < 9 || buf[0] != 27 || buf[1] != '[' || buf[2] != '<' {
		return Event{}, fmt.Errorf("not SGR mouse format")
	}

	i := 3
	button, next, ok := parseDecimal(buf, i)
	if !ok {
		return Event{}, fmt.Errorf("invalid SGR mouse button")
	}
	i = next
	if i >= len(buf) || buf[i] != ';' {
		return Event{}, fmt.Errorf("invalid SGR mouse separator")
	}
	i++

	x, next, ok := parseDecimal(buf, i)
	if !ok {
		return Event{}, fmt.Errorf("invalid SGR mouse x")
	}
	i = next
	if i >= len(buf) || buf[i] != ';' {
		return Event{}, fmt.Errorf("invalid SGR mouse separator")
	}
	i++

	y, next, ok := parseDecimal(buf, i)
	if !ok {
		return Event{}, fmt.Errorf("invalid SGR mouse y")
	}
	i = next
	if i >= len(buf) {
		return Event{}, fmt.Errorf("no SGR terminator found")
	}

	press := false
	switch buf[i] {
	case 'M':
		press = true
	case 'm':
		press = false
	default:
		return Event{}, fmt.Errorf("invalid SGR mouse terminator")
	}

	var mouseButton MouseButton
	switch button & 3 {
	case 0:
		mouseButton = MouseLeft
	case 1:
		mouseButton = MouseMiddle
	case 2:
		mouseButton = MouseRight
	}

	if button >= 64 {
		if button&1 != 0 {
			mouseButton = MouseWheelDown
		} else {
			mouseButton = MouseWheelUp
		}
	}

	return Event{Type: EventMouse, Button: mouseButton, X: x - 1, Y: y - 1, Press: press}, nil
}

func parseDecimal(buf []byte, idx int) (value, next int, ok bool) {
	if idx >= len(buf) {
		return 0, idx, false
	}
	start := idx
	val := 0
	for idx < len(buf) {
		b := buf[idx]
		if b < '0' || b > '9' {
			break
		}
		val = val*10 + int(b-'0')
		idx++
	}
	if idx == start {
		return 0, idx, false
	}
	return val, idx, true
}

func parseInput(buf []byte) (Event, error) {
	if len(buf) == 0 {
		return Event{}, fmt.Errorf("no input")
	}

	ch := buf[0]

	if ch == 27 { // ESC
		if len(buf) == 1 {
			return Event{Type: EventKey, Key: KeyEscape}, nil
		}
		if len(buf) >= 6 && buf[1] == '[' && buf[2] == '<' {
			if evt, err := parseSGRMouse(buf); err == nil {
				return evt, nil
			}
		}
		if len(buf) >= 3 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return Event{Type: EventKey, Key: KeyArrowUp}, nil
			case 'B':
				return Event{Type: EventKey, Key: KeyArrowDown}, nil
			case 'C':
				return Event{Type: EventKey, Key: KeyArrowRight}, nil
			case 'D':
				return Event{Type: EventKey, Key: KeyArrowLeft}, nil
			case 'H':
				return Event{Type: EventKey, Key: KeyHome}, nil
			case 'F':
				return Event{Type: EventKey, Key: KeyEnd}, nil
			case '1':
				if len(buf) >= 4 && buf[3] == '~' {
					return Event{Type: EventKey, Key: KeyHome}, nil
				}
			case '3':
				if len(buf) >= 4 && buf[3] == '~' {
					return Event{Type: EventKey, Key: KeyDelete}, nil
				}
			case '5':
				if len(buf) >= 4 && buf[3] == '~' {
					return Event{Type: EventKey, Key: KeyPageUp}, nil
				}
			case '6':
				if len(buf) >= 4 && buf[3] == '~' {
					return Event{Type: EventKey, Key: KeyPageDown}, nil
				}
			case 'M':
				if len(buf) >= 6 {
					return parseMouseEvent(buf[3:6])
				}
			}
			if len(buf) >= 5 && buf[2] == '1' {
				switch buf[3] {
				case '1', '2', '3', '4', '5':
					if buf[4] == '~' {
						return Event{Type: EventKey, Key: Key(int(KeyF1) + int(buf[3]-'1'))}, nil
					}
				}
			}
		}
		return Event{Type: EventKey, Key: KeyEscape}, nil
	}

	switch ch {
	case 1:
		return Event{Type: EventKey, Key: KeyCtrlA}, nil
	case 3:
		return Event{Type: EventKey, Key: KeyCtrlC}, nil
	case 4:
		return Event{Type: EventKey, Key: KeyCtrlD}, nil
	case 5:
		return Event{Type: EventKey, Key: KeyCtrlE}, nil
	case 9:
		return Event{Type: EventKey, Key: KeyTab}, nil
	case 11:
		return Event{Type: EventKey, Key: KeyCtrlK}, nil
	case 13:
		return Event{Type: EventKey, Key: KeyEnter}, nil
	case 21:
		return Event{Type: EventKey, Key: KeyCtrlU}, nil
	case 23:
		return Event{Type: EventKey, Key: KeyCtrlW}, nil
	case 127:
		return Event{Type: EventKey, Key: KeyBackspace}, nil
	default:
		return Event{Type: EventKey, Ch: rune(ch)}, nil
	}
}

func parseMouseEvent(buf []byte) (Event, error) {
	if len(buf) < 3 {
		return Event{}, fmt.Errorf("incomplete mouse event")
	}

	b := buf[0] - 32
	x := int(buf[1]) - 32
	y := int(buf[2]) - 32

	var button MouseButton
	switch b & 3 {
	case 0:
		button = MouseLeft
	case 1:
		button = MouseMiddle
	case 2:
		button = MouseRight
	}

	if b&64 != 0 {
		if b&1 != 0 {
			button = MouseWheelDown
		} else {
			button = MouseWheelUp
		}
	}

	return Event{Type: EventMouse, Button: button, X: x, Y: y, Press: true}, nil
}

func EnableMouse() {
	if !term.mouseEnabled {
		writeString(EnableMouseMode)
		term.mouseEnabled = true
	}
}

func DisableMouse() {
	if term.mouseEnabled {
		writeString(DisableMouseMode)
		term.mouseEnabled = false
	}
}

func EnableBracketedPaste() {
	if !term.pasteEnabled {
		writeString(EnableBracketPaste)
		term.pasteEnabled = true
	}
}

func DisableBracketedPaste() {
	if term.pasteEnabled {
		writeString(DisableBracketPaste)
		term.pasteEnabled = false
	}
}

func SetColor(fg, bg int) {
	term.currentFg = fg
	term.currentBg = bg
}

func SetAttr(bold, italic, underline, reverse bool) {
	term.currentBold = bold
	term.currentItalic = italic
	term.currentUnder = underline
	term.currentRev = reverse
}

func ResetAttr() {
	term.currentBold = false
	term.currentItalic = false
	term.currentUnder = false
	term.currentRev = false
	term.currentFg = 7
	term.currentBg = 0
}

func Size() (width, height int) {
	return term.width, term.height
}

func Flush() {
	Present()
}

func Fill(x, y, w, h int, ch rune) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			SetCell(x+dx, y+dy, ch, term.currentFg, term.currentBg)
		}
	}
}

func PrintAt(x, y int, text string) {
	for i, ch := range text {
		SetCell(x+i, y, ch, term.currentFg, term.currentBg)
	}
}

func Box(x, y, w, h int) {
	if w < 2 || h < 2 {
		return
	}

	SetCell(x, y, BoxTopLeft, term.currentFg, term.currentBg)
	SetCell(x+w-1, y, BoxTopRight, term.currentFg, term.currentBg)
	SetCell(x, y+h-1, BoxBottomLeft, term.currentFg, term.currentBg)
	SetCell(x+w-1, y+h-1, BoxBottomRight, term.currentFg, term.currentBg)

	for i := 1; i < w-1; i++ {
		SetCell(x+i, y, BoxHorizontal, term.currentFg, term.currentBg)
		SetCell(x+i, y+h-1, BoxHorizontal, term.currentFg, term.currentBg)
	}

	for i := 1; i < h-1; i++ {
		SetCell(x, y+i, BoxVertical, term.currentFg, term.currentBg)
		SetCell(x+w-1, y+i, BoxVertical, term.currentFg, term.currentBg)
	}
}

func ClearLineToEOL(y int) {
	ClearLine(y)
}

func ClearRegion(x, y, w, h int) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			SetCell(x+dx, y+dy, ' ', 7, 0)
		}
	}
}

func SaveCursorPos() {
	writeString(SaveCursor)
}

func RestoreCursorPos() {
	writeString(RestoreCursor)
}

func SetCursorVisible(visible bool) {
	if visible != term.cursorVisible {
		term.cursorVisible = visible
		if visible {
			writeString(ShowCursor)
		} else {
			writeString(HideCursor)
		}
	}
}

func IsRawMode() bool {
	return term.isRaw
}

func Bell() {
	writeString(BEL)
}

func Suspend() {
	if !term.initialized {
		return
	}

	disableRawMode()
	term.isRaw = false

	writeString(ClearScreen)
	writeString(ShowCursor)
	writeString(NormalScreen)

	syscall.Kill(syscall.Getpid(), syscall.SIGTSTP)
}

func Resume() {
	if !term.initialized {
		return
	}

	enableRawMode()
	term.isRaw = true

	writeString(AlternateScreen)
	if !term.cursorVisible {
		writeString(HideCursor)
	}
	writeString(ClearScreen)

	for y := 0; y < term.height; y++ {
		for x := 0; x < term.width; x++ {
			term.buffer.Cells[y][x].Dirty = true
		}
	}
}

func GetCursorPos() (x, y int) {
	if !term.initialized {
		return 0, 0
	}

	writeString(QueryCursorPos)

	var buf [32]byte
	fd := int(syscall.Stdin)

	fdSet := &syscall.FdSet{}
	fdSet.Bits[fd/64] |= 1 << (uint(fd) % 64)
	tv := syscall.Timeval{Sec: 1, Usec: 0} // 1 second timeout

	n, err := syscall.Select(fd+1, fdSet, nil, nil, &tv)
	if err != nil || n == 0 {
		return 0, 0
	}

	n, err = syscall.Read(syscall.Stdin, buf[:])
	if err != nil || n < 6 {
		return 0, 0
	}

	// Parse response: \x1b[row;colR
	response := string(buf[:n])
	if len(response) >= 6 && response[0] == '\x1b' && response[1] == '[' {
		var row, col int
		if _, err := fmt.Sscanf(response[2:], "%d;%dR", &row, &col); err == nil {
			return col - 1, row - 1 // Convert to 0-based
		}
	}

	return 0, 0
}

func HLine(x, y, length int, ch rune) {
	for i := 0; i < length; i++ {
		if x+i < term.width {
			SetCell(x+i, y, ch, term.currentFg, term.currentBg)
		}
	}
}

func VLine(x, y, length int, ch rune) {
	for i := 0; i < length; i++ {
		if y+i < term.height {
			SetCell(x, y+i, ch, term.currentFg, term.currentBg)
		}
	}
}

func DrawBytes(x, y int, data []byte) {
	for i, b := range data {
		if x+i < term.width && x+i >= 0 {
			SetCell(x+i, y, rune(b), term.currentFg, term.currentBg)
		}
	}
}

func ClearRect(x, y, w, h int) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			if x+dx >= 0 && x+dx < term.width && y+dy >= 0 && y+dy < term.height {
				SetCell(x+dx, y+dy, ' ', 7, term.currentBg)
			}
		}
	}
}

func SetCursor(x, y int) {
	term.cursorX = x
	term.cursorY = y
}

func HideCursorFunc() {
	SetCursorVisible(false)
}

func ShowCursorFunc() {
	SetCursorVisible(true)
}

func SetCursorStyle(style int) {
	term.cursorStyle = style
	writeString(fmt.Sprintf(ESC+"[%d q", style))
}

func EnableMouseFunc() {
	EnableMouse()
}

func DisableMouseFunc() {
	DisableMouse()
}

func SetInputMode(escDelay int) {
	term.escDelay = escDelay
}

func FlushInput() {
	fd := int(syscall.Stdin)
	flags, _, _ := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), F_GETFL, 0)
	syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), F_SETFL, flags|O_NONBLOCK)
	var buf [1024]byte
	for {
		_, err := syscall.Read(syscall.Stdin, buf[:])
		if err != nil {
			break
		}
	}
	syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), F_SETFL, flags)
}

func SaveBuffer() {
	needRealloc := term.savedBuffer == nil ||
		len(term.savedBuffer) != term.height ||
		(len(term.savedBuffer) > 0 && len(term.savedBuffer[0]) != term.width)

	if needRealloc {
		term.savedBuffer = make([][]Cell, term.height)
		for i := range term.savedBuffer {
			term.savedBuffer[i] = make([]Cell, term.width)
		}
	}

	for y := 0; y < term.height; y++ {
		for x := 0; x < term.width; x++ {
			if y < len(term.buffer.Cells) && x < len(term.buffer.Cells[y]) {
				term.savedBuffer[y][x] = term.buffer.Cells[y][x]
			}
		}
	}
}

func RestoreBuffer() {
	if term.savedBuffer == nil {
		return
	}

	for y := 0; y < term.height && y < len(term.savedBuffer); y++ {
		for x := 0; x < term.width && x < len(term.savedBuffer[y]); x++ {
			if y < len(term.buffer.Cells) && x < len(term.buffer.Cells[y]) {
				term.buffer.Cells[y][x] = term.savedBuffer[y][x]
				term.buffer.Cells[y][x].Dirty = true
			}
		}
	}
}

func GetCell(x, y int) (ch rune, fg, bg int) {
	if x < 0 || x >= term.width || y < 0 || y >= term.height {
		return ' ', 7, 0
	}

	cell := term.buffer.Cells[y][x]
	return cell.Ch, cell.Fg, cell.Bg
}

func Scroll(lines int) {
	if lines == 0 {
		return
	}

	if lines > 0 {
		for y := term.height - 1; y >= lines; y-- {
			for x := 0; x < term.width; x++ {
				term.buffer.Cells[y][x] = term.buffer.Cells[y-lines][x]
				term.buffer.Cells[y][x].Dirty = true
			}
		}

		for y := 0; y < lines && y < term.height; y++ {
			for x := 0; x < term.width; x++ {
				term.buffer.Cells[y][x] = Cell{Ch: ' ', Fg: 7, Bg: 0, Dirty: true}
			}
		}
	} else {
		lines = -lines
		for y := 0; y < term.height-lines; y++ {
			for x := 0; x < term.width; x++ {
				term.buffer.Cells[y][x] = term.buffer.Cells[y+lines][x]
				term.buffer.Cells[y][x].Dirty = true
			}
		}

		for y := term.height - lines; y < term.height; y++ {
			for x := 0; x < term.width; x++ {
				term.buffer.Cells[y][x] = Cell{Ch: ' ', Fg: 7, Bg: 0, Dirty: true}
			}
		}
	}
}

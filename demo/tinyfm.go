package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	tb "tinybox-example/tinybox"
)

type entry struct {
	name string
	dir  bool
}

type state struct {
	path   string
	items  []entry
	sel    int
	scroll int
	msg    string
}

func main() {
	st := &state{path: startPath()}
	st.reload()
	if err := tb.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "tinyfm:", err)
		return
	}
	defer tb.Close()
	tb.SetCursorVisible(false)
	for {
		draw(st)
		evt, err := tb.PollEvent()
		if err != nil {
			st.msg = err.Error()
			continue
		}
		if evt.Type != tb.EventKey {
			continue
		}
		switch evt.Key {
		case tb.KeyCtrlC, tb.KeyEscape:
			return
		case tb.KeyArrowUp:
			st.move(-1)
		case tb.KeyArrowDown:
			st.move(1)
		case tb.KeyEnter:
			st.openSelection()
		case tb.KeyBackspace:
			st.up()
		default:
			if evt.Ch == 'q' || evt.Ch == 'Q' {
				return
			}
		}
	}
}

func startPath() string {
	if len(os.Args) > 1 {
		if abs, err := filepath.Abs(os.Args[1]); err == nil {
			return abs
		}
	}
	if pwd, err := os.Getwd(); err == nil {
		return pwd
	}
	return "."
}

func (s *state) reload() {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		s.items = nil
		s.msg = err.Error()
		return
	}
	list := make([]entry, 0, len(entries))
	for _, e := range entries {
		list = append(list, entry{name: e.Name(), dir: e.IsDir()})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].dir == list[j].dir {
			return list[i].name < list[j].name
		}
		return list[i].dir
	})
	s.items = list
	if s.sel >= len(list) {
		s.sel = len(list) - 1
	}
	if s.sel < 0 {
		s.sel = 0
	}
	if len(list) == 0 {
		s.sel = 0
	}
	s.msg = ""
}

func (s *state) move(step int) {
	if len(s.items) == 0 {
		s.sel = 0
		return
	}
	s.sel += step
	if s.sel < 0 {
		s.sel = 0
	} else if s.sel >= len(s.items) {
		s.sel = len(s.items) - 1
	}
}

func (s *state) openSelection() {
	if len(s.items) == 0 {
		return
	}
	it := s.items[s.sel]
	next := filepath.Join(s.path, it.name)
	if it.dir {
		s.path = next
		s.sel, s.scroll = 0, 0
		s.reload()
		return
	}
	s.msg = next
}

func (s *state) up() {
	parent := filepath.Dir(s.path)
	if parent == s.path {
		return
	}
	s.path = parent
	s.sel, s.scroll = 0, 0
	s.reload()
}

func draw(s *state) {
	tb.Clear()
	w, h := tb.Size()
	if w <= 0 || h <= 2 {
		tb.Flush()
		return
	}
	tb.SetColor(14, 0)
	tb.PrintAt(0, 0, " "+s.path)
	tb.SetColor(8, 0)
	tb.PrintAt(0, 1, " arrows navigate · enter open · backspace up · q quit")
	viewTop := 2
	view := h - viewTop - 1
	if view < 1 {
		view = 1
	}
	if s.sel < s.scroll {
		s.scroll = s.sel
	}
	if s.sel >= s.scroll+view {
		s.scroll = s.sel - view + 1
	}
	if max := len(s.items) - view; max >= 0 && s.scroll > max {
		s.scroll = max
	}
	for i := 0; i < view && s.scroll+i < len(s.items); i++ {
		idx := s.scroll + i
		it := s.items[idx]
		y := viewTop + i
		if idx == s.sel {
			tb.SetColor(0, 12)
		} else if it.dir {
			tb.SetColor(11, 0)
		} else {
			tb.SetColor(15, 0)
		}
		tb.PrintAt(0, y, formatEntry(it))
	}
	tb.SetColor(8, 0)
	status := s.msg
	if status == "" {
		status = fmt.Sprintf("%d item(s)", len(s.items))
	}
	tb.PrintAt(0, h-1, " "+status)
	tb.Flush()
}

func formatEntry(e entry) string {
	name := e.name
	if e.dir {
		name += "/"
	}
	return " " + name
}

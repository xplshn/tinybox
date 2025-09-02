# tinybox - Minimalist TUI Library

I needed a simple way to build terminal interfaces in Go. Everything out there was either massive (tcell), abandoned (termbox), or tried to force some architecture on me (bubbletea with its Elm thing). I just wanted to draw stuff on the screen and read keyboard input without pulling in half the internet.

So I wrote tinybox. It's one Go file, about 1000 lines. You can read the whole thing in an afternoon. Copy it into your project and modify it however you want. No dependencies, no build systems, no package managers.

## What It Does

Tinybox gives you raw terminal access. You get a grid of cells, you put characters in them, you call Present() to update the screen. That's the core of it. 

It handles the annoying parts like entering raw mode, parsing escape sequences, tracking what changed so you're not redrawing everything constantly. Mouse events work. Colors work. You can catch Ctrl-Z properly. The stuff you'd expect.

The API is deliberately small. Init() to start, Close() to cleanup, SetCell() to draw, PollEvent() to read input. Maybe 30 functions total. If you need something that's not there, the code is right there - add it yourself.

## How It Works

The terminal is just a 2D grid of cells. Each cell has a character, foreground color, background color, and some attributes like bold or underline. Box maintains two buffers - what you're drawing to, and what's currently on screen. When you call Present(), it figures out what changed and sends only those updates to the terminal.

Input comes through as events - keyboard, mouse, resize. The event loop is yours to write. Tinybox just gives you the events, you decide what to do with them. No callbacks, no handlers, no framework nonsense.

Here's the simplest possible program:

```go
tui.Init()
defer tui.Close()
tui.PrintAt(0, 0, "some string")
tui.Present()
tui.PollEvent()  // wait for key
```

## Example

Please refer to example.go, it contains a minimal program to fetch some Systemdata and displays said data with tinybox.

```
make
```
```
./example
```
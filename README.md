# Tinybox

I needed a simple way to build terminal interfaces in Go. Everything out there was either massive (tcell), abandoned (termbox), or tried to force some architecture on me (bubbletea with its Elm thing). I just wanted to draw stuff on the screen and read keyboard input without pulling in half the internet.

So I wrote Tinybox. It's one Go file, about 1000 lines. You can read the whole thing in an afternoon. Copy it into your project and modify it however you want. No dependencies, no build systems, no package managers.

<div align="center">
<tr>
<td><a href="https://github.com/user-attachments/assets/d1e53971-32e8-4d88-94c2-49b31d1255ca"><img width="300" alt="CGA" src="https://github.com/user-attachments/assets/d1e53971-32e8-4d88-94c2-49b31d1255ca" /></a></td>
<td><a href="https://github.com/user-attachments/assets/a7d4efb5-66b1-47e6-adc8-46b38fd8feb0"><img width="300" alt="Boxes" src="https://github.com/user-attachments/assets/a7d4efb5-66b1-47e6-adc8-46b38fd8feb0" /></a></td>
</tr>
</table>
</div>

## What It Does

Tinybox gives you raw terminal access which means you get a grid of cells, you put characters in them, you call Present() to update the screen. That's basically the core of it.

It handles the annoying parts like entering raw mode, parsing escape sequences, tracking what changed so you're not redrawing everything constantly. Mouse events work. Colors work. You can catch Ctrl-Z properly. The stuff you'd expect.

The API is deliberately small. Init() to start, Close() to cleanup, SetCell() to draw, PollEvent() to read input. Maybe 30 functions total. If you need something that's not there, the code is right there - so you can simply add it yourself.

## How It Works

The terminal is just a 2D grid of cells. Each cell has a character, foreground color, background color, and some attributes like bold or underline. Tinybox maintains two buffers - what you're drawing to, and what's currently on screen. When you call Present(), it figures out what changed and sends only those updates to the terminal.

Input comes through as events - keyboard, mouse, resize. The event loop is yours to write. Tinybox just gives you the events, you decide what to do with them. No callbacks, no handlers, no framework nonsense.

Here's the simplest possible program:

```go
tb.Init()
defer tb.Close()
tb.PrintAt(0, 0, "some string")
tb.Present()
tb.PollEvent()  // wait for key
```
Look at example.go if you want to see something more complex. It's a basic system monitor that shows how to handle resize, use colors, and create a simple table layout (screenshot). That sample leans on Linux's statfs fields; tweak the disk bits if you're building it on OpenBSD or the other BSDs.
The API won't change because there's no version to track. You have the code. If you need it to work differently, change it.

There’s also a tiny demo app under `demo/` if you want to poke at the API without writing boilerplate. It uses the same primitives (draw cells, poll events, toggle mouse) and nothing more.

## Example
```
make
```
```
./example
```
## Design Choices

The code reads top to bottom - constants, types, low-level terminal stuff, then the public API. 

It sticks to the plain POSIX termios/ioctl calls. The `termios_*.go` pair only map native ioctl constants, `select_*.go` hides the syscall differences, and `fdset_posix.go` flips the right bits so `select` works the same everywhere.

Two background goroutines handle signals; one for terminal resize (SIGWINCH) and one for resume from suspension (SIGCONT). These run in the background but don't do any terminal drawing, just update internal state and queue events. The main program loop is single-threaded.
No Unicode normalization or grapheme clustering or any of that. The terminal handles displaying Unicode, we just pass it through.

### Colors 

Colors use the 256-color palette because that's what every modern terminal supports.

## What's Included

The basics are all there, which means you can draw text anywhere on screen with colors and attributes. Tinybox drawing characters work for making borders and tables, mouse support includes clicks, drags, and scroll wheels. Terminal resize is handled automatically.
The dirty cell tracking means it's fast even over SSH. Only the cells that changed get updated. You can build surprisingly complex interfaces and they'll still feel snappy.
Suspend/resume works properly. Hit Ctrl-Z, do something in the shell, fg back, and your program continues where it left off. The terminal state is saved and restored correctly.
There's a buffer save/restore mechanism if you need to draw temporary overlays like menus or dialogs. Save the buffer, draw your popup, then restore when done.

## What's Not Included

No widget library. No buttons, text fields, or scroll views. Those are your problem. Tinybox gives you a canvas and input events - what you build is up to you.
No configuration files. No themes. No plugins. If you want different defaults, change the source.
No layout managers. You calculate where things go. It's not that hard.
No documentation beyond this README and the code itself. The function names are clear and the implementation is right there if you need details.

## Why Bother?

Sometimes you need a TUI and don't want to deal with ncurses or massive Go libraries. Sometimes you want code you can actually understand. Sometimes smaller is better.
Tinybox does exactly what it needs to do and nothing more. It's not trying to be everything for everyone.
If you need more, there are plenty of feature-rich alternatives. If you want less, you probably don't want a TUI at all.

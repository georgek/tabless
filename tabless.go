// tabless - a table viewer like less
// Copyright (C) 2018 George Kettleborough

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/alexflint/go-arg"
)

const (
	maxInt        = int(^uint(0)>>1)
	readAheadMult = 10
)

// isNumeric returns true if the string is considered a float by
// strconv.ParseFloat
func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

type tabless struct {
	// the application's screen
	screen tcell.Screen

	// the main primitive
	table *tview.Table

	// the input file
	file *os.File

	// channel for sending rows of cells to be added to table
	cellCh chan []string

	// channel to request more rows to be added to table
	reqCh chan int

	// channel to request redraw
	drawCh chan bool

	// number of fixed rows and columns
	rfix, cfix int

	// table borders
	borders bool

	// records whether we have a full screen full yet
	screenFull bool
}

func newTabless() *tabless {
	return &tabless{}
}

func (t *tabless) read(delimiter string) {

	scanner := bufio.NewScanner(t.file)

	for scanner.Scan() {
		line := scanner.Text()
		texts := strings.Split(line, delimiter)
		t.cellCh <- texts
	}
	close(t.cellCh)
}

// add_cells takes a Cell from cellCh and adds it to the table
func (t *tabless) addRows() {
	row, reqRows := 0, 0
	for {
		reqRows = max(<-t.reqCh, reqRows)
		for row < reqRows {
			select {
			case newReq, more := <-t.reqCh:
				if !more {
					return
				}
				reqRows = max(newReq, reqRows)
			case cells, more := <-t.cellCh:
				if !more {
					reqRows = row
					break
				}
				for col, text := range cells {
					alignment := tview.AlignCenter
					expansion := 1
					maxWidth := 10
					color := tcell.ColorWhite
					if col < t.cfix || row < t.rfix {
						color = tcell.ColorYellow
					} else if isNumeric(text) {
						alignment = tview.AlignRight
						maxWidth = 20
					} else {
						alignment = tview.AlignLeft
						expansion = 2
					}

					t.table.SetCell(row, col,
						tview.NewTableCell(text).
							SetTextColor(color).
							SetAlign(alignment).
							SetMaxWidth(maxWidth).
							SetExpansion(expansion))
				}
				if !t.screenFull {
					select {
					case t.drawCh <- true:
						break
					default:
						break
					}
				}
				row++
			}
		}
		// send to unblock event loop for redraw
		t.drawCh <- true
		t.screenFull = true
	}
}

// convert some extra keys into standard tview.Table bindings
func inputCapture (event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	switch key {
	case tcell.KeyCtrlN:
		return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	case tcell.KeyCtrlP:
		return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	case tcell.KeyCtrlV:
		return tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone)
	case tcell.KeyRune:
		rune := event.Rune()
		if (event.Modifiers() & tcell.ModAlt) != 0 {
			switch rune {
			case 'v':
				// M-v
				return tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone)
			case '>':
				// M->
				return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
			case '<':
				// M-<
				return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
			default:
				return event
			}
		} else {
			switch rune {
			case 'q':
				// app.Stop()
				return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
			default:
				return event
			}
		}
	default:
		return event
	}
}

func (t *tabless) Draw() {
	if t.screen == nil {
		return
	}

	t.table.Draw(t.screen)

	t.screen.Show()
}

func (t *tabless) Run() error {
	var err error

	t.reqCh = make(chan int)
	t.cellCh = make(chan []string)
	t.drawCh = make(chan bool)

	t.screen, err = tcell.NewScreen()
	if err != nil {
		return err
	}
	if err = t.screen.Init(); err != nil {
		return err
	}

	// We catch panics to clean up because they mess up the terminal.
	defer func() {
		if p := recover(); p != nil {
			if t.screen != nil {
				t.screen.Fini()
			}
			panic(p)
		}
	}()

	t.table = tview.NewTable().SetBorders(t.borders)
	t.table.Select(0, 0).SetFixed(t.rfix, t.cfix)
	width, height := t.screen.Size()
	t.table.SetRect(0, 0, width, height)
	// request a screenful of rows
	var rowOffset int
	rowOffset = 0
	if t.borders {
		t.reqCh <- rowOffset + height/2
	} else {
		t.reqCh <- rowOffset + height
	}

	waiting, done := false, false
	go func() {
		for {
			if waiting {
				done = true
				waiting = false
			}
			<-t.drawCh
			t.Draw()
		}
	}()

	// event loop
	for {
		if t.screen == nil {
			break
		}

		event := t.screen.PollEvent()
		if event == nil {
			break
		}
		switch event := event.(type) {
		case *tcell.EventKey:
			event = inputCapture(event)

			if event.Key() == tcell.KeyEnd && !done {
				t.reqCh <- maxInt
				waiting = true
				continue
			} else if event.Key() == tcell.KeyCtrlC {
				if waiting {
					waiting = false
					done = true
					t.file.Close()
					continue
				}
				t.Stop()
				return nil
			} else if waiting {
				continue
			}

			// passthrough to the table
			handler := t.table.InputHandler()
			handler(event, func(p tview.Primitive) {})

		case *tcell.EventResize:
			width, height = t.screen.Size()
			t.table.SetRect(0, 0, width, height)
			t.screen.Clear()
			t.Draw()
		}

		rowOffset, _ = t.table.GetOffset()
		if t.borders {
			t.reqCh <- rowOffset + height/2
		} else {
			t.reqCh <- rowOffset + height
		}
		t.drawCh <- true
	}

	return nil
}

func (t *tabless) Stop() {
	if t.screen == nil {
		return
	}
	close(t.reqCh)
	t.screen.Fini()
	t.screen = nil
}

func main() {

	var args struct {
		Delimiter string `arg:"-d" help:"Column delimiter character(s)"`
		Borders   bool   `arg:"-b" help:"Display table with borders"`
		Input     string `arg:"positional" help:"Input file"`
		Cfix      int    `arg:"-c" help:"Number of columns to fix"`
		Rfix      int    `arg:"-r" help:"Number of rows to fix"`
	}
	args.Delimiter = "\t"
	args.Borders = true
	args.Cfix = 0
	args.Rfix = 1

	parser := arg.MustParse(&args)

	tabless := newTabless()

	var err error
	if args.Input != "" && args.Input != "-" {
		tabless.file, err = os.Open(args.Input)
		if err != nil {
			parser.Fail("file not found")
		}
	} else {
		// make sure stdin is being piped
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			parser.WriteUsage(os.Stderr)
			os.Exit(1)
		} else {
			tabless.file = os.Stdin
		}
	}

	tabless.cfix, tabless.rfix = args.Cfix, args.Rfix
	tabless.borders = args.Borders

	go tabless.read(args.Delimiter)
	go tabless.addRows()

	if err := tabless.Run(); err != nil {
		panic(err)
	}
}

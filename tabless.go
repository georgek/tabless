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
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	MaxInt        = int(^uint(0)>>1)
	readAheadMult = 10
)

var (
	delimiter = flag.String("d", "\t", "column delimiter to use")
	borders   = flag.Bool("b", true, "display graphical borders")
)

type Cell struct{
	text string
	col, row int
}

// isNumeric returns true if the string is considered a float by
// strconv.ParseFloat
func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// read gets line from file and inserts cells into cell_ch. It waits until a
// number of rows are requested by passing an integer in the draw_req channel,
// then more of the file is read until the table has that size, or the end of
// the file is reached. The number of lines actually read so far is put back
// into the channel in response to every input on the channel.
func read(file *os.File, delimiter string, draw_ch chan int, cell_ch chan *Cell) {

	file_open := true
	scanner := bufio.NewScanner(file)

	var requested, row int
	row = 0
	for {
		requested = <-draw_ch
		if requested == 0 {
			file_open = false
			file.Close()
		}
		if !file_open {
			draw_ch <- row
			continue
		}
		put_rows := func(n int) {
			for ; row < n; row++ {
				if !scanner.Scan() {
					break
				}
				line := scanner.Text()
				texts := strings.Split(line, delimiter)
				for col, text := range texts {
					cell := new(Cell)
					cell.text, cell.col, cell.row = text, col, row
					cell_ch <- cell
				}
			}
		}
		put_rows(requested)
		draw_ch <- row
		put_rows(requested*readAheadMult)
	}
}

// add_cells takes a Cell from cell_ch and adds it to the table
func add_cells(table *tview.Table, cell_ch chan *Cell, rfix, cfix int) {
	var cell *Cell
	for {
		cell = <-cell_ch

		alignment := tview.AlignCenter
		expansion := 1
		max_width := 10
		color := tcell.ColorWhite
		if cell.col < cfix || cell.row < rfix {
			color = tcell.ColorYellow
		} else if isNumeric(cell.text) {
			alignment = tview.AlignRight
			max_width = 20
		} else {
			alignment = tview.AlignLeft
			expansion = 2
		}

		table.SetCell(cell.row, cell.col,
			tview.NewTableCell(cell.text).
				SetTextColor(color).
				SetAlign(alignment).
				SetMaxWidth(max_width).
				SetExpansion(expansion))
	}
}

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] [filename]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	var input_file *os.File
	var err error
	if flag.NArg() > 0 && flag.Arg(0) != "-" {
		input_file, err = os.Open(flag.Arg(0))
		if err != nil {
			panic(err)
		}
	} else {
		// check if stdin is being piped
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			flag.Usage()
			os.Exit(0)
		} else {
			input_file = os.Stdin
		}
	}

	cfix, rfix := 0, 1

	app := tview.NewApplication()
	table := tview.NewTable().SetBorders(*borders)

	// chan to request new rows are added
	draw_ch := make(chan int)
	// chan for new cells to be added
	cell_ch := make(chan *Cell)
	go read(input_file, *delimiter, draw_ch, cell_ch)
	go add_cells(table, cell_ch, rfix, cfix)

	table.Select(0, 0).SetFixed(rfix, cfix)
	app.SetRoot(table, true).SetFocus(table)

	// before drawing make sure enough rows have been added to the table
	app.SetBeforeDrawFunc(func (screen tcell.Screen) bool {
		_, screen_height := screen.Size()
		row_offset, _ := table.GetOffset()
		// try to read an extra screenful of rows
		var last_row int
		if *borders {
			last_row = row_offset + screen_height
		} else {
			last_row = row_offset + screen_height*2
		}

		draw_ch <- last_row
		<-draw_ch

		// returning false makes the table draw
		return false
	})

	app.SetInputCapture(func (event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()
		switch key {
		case tcell.KeyEnd:
			draw_ch <- MaxInt
			<-draw_ch
			return event
		case tcell.KeyCtrlN:
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case tcell.KeyCtrlP:
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		case tcell.KeyCtrlV:
			return tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone)
		case tcell.KeyRune:
			rune := event.Rune()
			switch rune {
			case 'q':
				app.Stop()
				return nil
			case 'v':
				if (event.Modifiers() & tcell.ModAlt) != 0 {
					// Alt-V
					return tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone)
				} else {
					return event
				}
			default:
				return event
			}
		default:
			return event
		}
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}

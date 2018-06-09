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

func max(x int, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}

type Tabless struct {
	// the application's screen
	screen tcell.Screen

	// the main primitive
	table *tview.Table

	// the input file
	file *os.File

	// channel for sending rows of cells to be added to table
	cell_ch chan []string

	// channel to request more rows to be added to table
	draw_ch chan int

	// number of fixed rows and columns
	rfix, cfix int

	// table borders
	borders bool

	// records whether we have a full screen full yet
	screenFull bool
}

func NewTabless() *Tabless {
	return &Tabless{}
}

func (t *Tabless) read(delimiter string) {

	scanner := bufio.NewScanner(t.file)

	for scanner.Scan() {
		line := scanner.Text()
		texts := strings.Split(line, delimiter)
		t.cell_ch <- texts
	}
	close(t.cell_ch)
}

// add_cells takes a Cell from cell_ch and adds it to the table
func (t *Tabless) add_rows() {
	row, req_rows := 0, 0
	for {
		req_rows = max(<-t.draw_ch, req_rows)
		for row < req_rows {
			select {
			case new_req, more := <-t.draw_ch:
				if !more {
					return
				}
				req_rows = max(new_req, req_rows)
			case cells, more := <-t.cell_ch:
				if !more {
					req_rows = row
					break
				}
				for col, text := range cells {
					alignment := tview.AlignCenter
					expansion := 1
					max_width := 10
					color := tcell.ColorWhite
					if col < t.cfix || row < t.rfix {
						color = tcell.ColorYellow
					} else if isNumeric(text) {
						alignment = tview.AlignRight
						max_width = 20
					} else {
						alignment = tview.AlignLeft
						expansion = 2
					}

					t.table.SetCell(row, col,
						tview.NewTableCell(text).
							SetTextColor(color).
							SetAlign(alignment).
							SetMaxWidth(max_width).
							SetExpansion(expansion))
				}
				if !t.screenFull {
					t.screen.PostEvent(tcell.NewEventInterrupt(nil))
				}
				row++
			}
		}
		// send to unblock event loop for redraw
		t.screen.PostEventWait(tcell.NewEventInterrupt(nil))
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
		switch rune {
		case 'q':
			// app.Stop()
			return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
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
}

func (t *Tabless) Draw() {
	if t.screen == nil {
		return
	}
	width, height := t.screen.Size()
	t.table.SetRect(0, 0, width, height)

	t.table.Draw(t.screen)

	t.screen.Show()
}

func (t *Tabless) Run() error {
	var err error

	t.draw_ch = make(chan int)
	t.cell_ch = make(chan []string)

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
	// request a screenful of rows
	var screen_height, row_offset int
	_, screen_height = t.screen.Size()
	row_offset = 0
	if t.borders {
		t.draw_ch <- row_offset + screen_height/2
	} else {
		t.draw_ch <- row_offset + screen_height
	}

	t.Draw()

	// event loop
	waiting := false
	for {
		if t.screen == nil {
			break
		}

		event := t.screen.PollEvent()
		if event == nil {
			break
		}
		switch event := event.(type) {
		case *tcell.EventInterrupt:
			waiting = false
			t.Draw()
			continue
		case *tcell.EventKey:
			event = inputCapture(event)

			if event.Key() == tcell.KeyEnd {
				t.draw_ch <- MaxInt
				waiting = true
			} else if event.Key() == tcell.KeyCtrlC {
				if waiting {
					t.file.Close()
					continue
				}
				t.Stop()
				return nil
			} else if event.Key() == tcell.KeyDown {
				// stop passing down if we're already at end of table
				// if t.table.GetRowCount()*2 - screen_height - row_offset <= 0 {
				// 	break
				// }
			}

			// passthrough to the table
			handler := t.table.InputHandler()
			handler(event, func(p tview.Primitive) {})

		case *tcell.EventResize:
			t.screen.Clear()
			t.Draw()
			_, screen_height = t.screen.Size()
		}

		row_offset, _ = t.table.GetOffset()
		if t.borders {
			t.draw_ch <- row_offset + screen_height/2
		} else {
			t.draw_ch <- row_offset + screen_height
		}
	}

	return nil
}

func (t *Tabless) Stop() {
	if t.screen == nil {
		return
	}
	close(t.draw_ch)
	t.screen.Fini()
	t.screen = nil
}

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] [filename]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	tabless := NewTabless()

	var err error
	if flag.NArg() > 0 && flag.Arg(0) != "-" {
		tabless.file, err = os.Open(flag.Arg(0))
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
			tabless.file = os.Stdin
		}
	}

	cfix, rfix := 0, 1

	tabless.cfix, tabless.rfix = cfix, rfix
	tabless.borders = *borders

	go tabless.read(*delimiter)
	go tabless.add_rows()

	if err := tabless.Run(); err != nil {
		panic(err)
	}
}

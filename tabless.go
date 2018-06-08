package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

type EventDone struct {
	t time.Time
}

func (ev *EventDone) When() time.Time {
	return ev.t
}

func NewEventDone() *EventDone {
	return &EventDone{t: time.Now()}
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

	// channel for sending rows of cells to be added to table
	cell_ch chan []string

	// channel to request more rows to be added to table
	draw_ch chan int

	// number of fixed rows and columns
	rfix, cfix int

	// table borders
	borders bool
}

func NewTabless() *Tabless {
	return &Tabless{}
}

func (t *Tabless) read(file *os.File, delimiter string) {

	scanner := bufio.NewScanner(file)

	row := 0
	for ; ; row++ {
		if !scanner.Scan() {
			close(t.cell_ch)
			return
		}
		line := scanner.Text()
		texts := strings.Split(line, delimiter)
		t.cell_ch <- texts
	}
	close(t.cell_ch)
}

// add_cells takes a Cell from cell_ch and adds it to the table
func (t *Tabless) add_rows() {
	row, max_rows := 0, 0
	for {
		max_rows = max(<-t.draw_ch, max_rows)
		for row < max_rows {
			select {
			case cells, more := <-t.cell_ch:
				if !more {
					max_rows = row
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
				row++
			case draw_req := <-t.draw_ch:
				max_rows = max(draw_req, max_rows)
			}
		}
		// send to unblock event loop for redraw
		t.screen.PostEventWait(NewEventDone())
		// non-blocking send to channel to enable other goroutines to
		// block on this
		select {
		case t.draw_ch <- row:
			break
		default:
			break
		}
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
	row_offset, _ = t.table.GetOffset()
	if t.borders {
		t.draw_ch <- row_offset + screen_height/2
	} else {
		t.draw_ch <- row_offset + screen_height
	}
	// wait for screenful until first draw
	<-t.draw_ch

	t.Draw()

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
		case *EventDone:
			t.Draw()
			continue
		case *tcell.EventKey:
			event = inputCapture(event)

			if event.Key() == tcell.KeyEnd {
				t.draw_ch <- MaxInt
			} else if event.Key() == tcell.KeyCtrlC {
				t.Stop()
				return nil
			}

			handler := t.table.InputHandler()
			handler(event, func(p tview.Primitive) {})

		case *tcell.EventResize:
			t.screen.Clear()
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

	tabless := NewTabless()
	tabless.cfix, tabless.rfix = cfix, rfix
	tabless.borders = *borders

	go tabless.read(input_file, *delimiter)
	go tabless.add_rows()

	if err := tabless.Run(); err != nil {
		panic(err)
	}
}

package main

import (
	"bufio"
	"os"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type Cell struct{
	text string
	i, j int
}

// read gets line from file and inserts cells into cell_ch. It waits until a
// number of rows are requested by passing an integer in the draw_req channel,
// then more of the file is read until the table has that size, or the end of
// the file is reached. The number of lines actually read so far is put back
// into the channel in response to every input on the channel.
func read(file *os.File, draw_ch chan int, cell_ch chan *Cell) {

	scanner := bufio.NewScanner(file)

	var requested int
	j := 0
	for {
		requested = <-draw_ch
		if requested == 0 {
			return
		}
		for ; scanner.Scan() && j < requested; j++ {
			line := scanner.Text()
			texts := strings.Split(line, "\t")
			for i, text := range texts {
				cell := new(Cell)
				cell.text, cell.i, cell.j = text, i, j
				cell_ch <- cell
			}
		}
		draw_ch <- j
	}
}

// add_cells takes a Cell from cell_ch and adds it to the table
func add_cells(table *tview.Table, cell_ch chan *Cell, rfix, cfix int) {
	var cell *Cell
	for {
		cell = <-cell_ch
		color := tcell.ColorWhite
		if cell.i < cfix || cell.j < rfix {
			color = tcell.ColorYellow
		}
		table.SetCell(cell.j, cell.i,
			tview.NewTableCell(cell.text).
				SetTextColor(color).
				SetAlign(tview.AlignCenter))
	}
}

func main() {

	borders := true
	cfix, rfix := 0, 1

	app := tview.NewApplication()
	table := tview.NewTable().SetBorders(borders)

	// chan to request new rows are added
	draw_ch := make(chan int)
	// chan for new cells to be added
	cell_ch := make(chan *Cell)
	go read(os.Stdin, draw_ch, cell_ch)
	go add_cells(table, cell_ch, rfix, cfix)

	table.Select(0, 0).SetFixed(rfix, cfix)
	app.SetRoot(table, true).SetFocus(table)

	// before drawing make sure enough rows have been added to the table
	app.SetBeforeDrawFunc(func (screen tcell.Screen) bool {
		_, screen_height := screen.Size()
		row_offset, _ := table.GetOffset()
		// try to read an extra screenful of rows
		var last_row int
		if borders {
			last_row = row_offset + screen_height
		} else {
			last_row = row_offset + screen_height*2
		}

		if table.GetRowCount() < last_row {
			draw_ch <- last_row
			<-draw_ch
		}
		// returning false makes the table draw
		return false
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}

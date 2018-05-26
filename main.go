package main

import (
	"bufio"
	"os"
	"strings"
	// "time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func run(app *tview.Application, table *tview.Table) {
	table.Select(0, 0).SetFixed(1, 1)

	if err := app.SetRoot(table, true).SetFocus(table).Run(); err != nil {
		panic(err)
	}
}

func read(app *tview.Application, table *tview.Table, file *os.File, sfull chan bool) {
	_, _, _, height := table.GetInnerRect()
	visibleRows := height / 2

	scanner := bufio.NewScanner(file)

	for j := 0; scanner.Scan(); j++ {
		line := scanner.Text()
		cells := strings.Split(line, "\t")
		for i, cell := range cells {
			color := tcell.ColorWhite
			if i < 1 || j < 1 {
				color = tcell.ColorYellow
			}
			table.SetCell(j, i,
				tview.NewTableCell(cell).
					SetTextColor(color).
					SetAlign(tview.AlignCenter))
		}
		if j == visibleRows*2 {
			sfull <- true
		} else if j > visibleRows*2 {
			// app.Draw()
		}
	}
}

func main() {
	app := tview.NewApplication()
	table := tview.NewTable().SetBorders(true)

	// indicates that a screen has been filled
	sfull := make(chan bool)
	go read(app, table, os.Stdin, sfull)

	// wait for screen full
	<-sfull

	run(app, table)
}

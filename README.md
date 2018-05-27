# Tabless #

A command line utility like *less* but for displaying tabular data
graphically. Like less, tabless does not need to read the entire file (or
pipe) before it displays it, so it's quick to use when working on the command
line.

This uses [tcell](https://github.com/gdamore/tcell) and
[tview](https://github.com/rivo/tview) to handle the drawing.

## Installation ##

``` shell
go get github.com/georgek/tabless
```

## Issues ##

When using the PgDn key tabless will attempt to read until the end of the
file. There is currently no way to abort this action so huge files or infinite
pipes will cause a hang.

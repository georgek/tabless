# tabless #

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

## Usage ##

``` shell
# read from file
tabless filename
# read from pipe
table_generator | tabless
```

## Utilities ##

Use the included `generate-massive-table.py` to generate a lot of data for
testing. Without a command line argument it generates a huge table so be
careful.

``` shell
# generate 10000 rows
python utils/generate-massive-table.py 10000 | tabless
```

## Issues ##

When using the PgDn key tabless will attempt to read until the end of the
file. There is currently no way to abort this action so huge files or infinite
pipes will cause a hang.

# tabless #

[![Build Status](https://travis-ci.org/georgek/tabless.svg?branch=master)](https://travis-ci.org/georgek/tabless)

A command line utility like `less` but for displaying tabular data
graphically. Like `less`, `tabless` does not need to read the entire file (or
pipe) before it displays it, so it's quick to use when working on the command
line.

This uses [tcell](https://github.com/gdamore/tcell) and
[tview](https://github.com/rivo/tview) to handle the drawing.

## Installation ##

``` shell
go get github.com/georgek/tabless
```

## Example ##

``` shell
# read from file
tabless filename
# read from pipe
table_generator | tabless
```

## Usage ##

``` shell
usage: tabless [flags] [filename]

flags:
  -b    display graphical borders (default true)
  -d string
        column delimiter to use (default "\t")
```

## Utilities ##

Use the included `generate-massive-table.py` to generate a lot of data for
testing. Without a command line argument it generates a huge table so be
careful.

``` shell
# generate 10000 rows
python utils/generate-massive-table.py -r 10000 | tabless
# simulate slow pipe (10 rows per second)
python utils/generate-massive-table.py -s 0.1 -r 10000 | tabless
```

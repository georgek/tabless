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
	"testing"
)

func TestMax(t *testing.T) {
	if got, want := max(2, 3), 3; got != want {
		t.Errorf("max got %d, want %d", got, want)
	}
}

func TestIsNumeric(t *testing.T) {
	if got, want := isNumeric("3.2"), true; got != want {
		t.Errorf("isNumeric got %t, want %t", got, want)
	}
	if got, want := isNumeric("3"), true; got != want {
		t.Errorf("isNumeric got %t, want %t", got, want)
	}
	if got, want := isNumeric("three"), false; got != want {
		t.Errorf("isNumeric got %t, want %t", got, want)
	}
}

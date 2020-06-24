// Copyright (C) 2020  Raziman

package main

import (
	"github.com/rivo/tview"
)

func main() {

	readConfig()

	app := tview.NewApplication()

	start(app)

}

// Copyright (C) 2020  Raziman

package main

import (
	"os"

	"github.com/rivo/tview"
)

func main() {

	os.Setenv("TEST", "false")

	args := getArgs()

	readConfig(args)

	app := tview.NewApplication()

	start(app, args)

}

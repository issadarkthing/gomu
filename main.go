// Copyright (C) 2020  Raziman

package main

import (
	"log"
	"os"
	"path"

	"github.com/rivo/tview"
)

func main() {
	setupLog()
	os.Setenv("TEST", "false")
	args := getArgs()

	app := tview.NewApplication()

	// main loop
	start(app, args)
}

func setupLog() {
	tmpDir := os.TempDir()
	logFile := path.Join(tmpDir, "gomu.log")
	file, e := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if e != nil {
		log.Fatalf("Error opening file %s", logFile)
	}

	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

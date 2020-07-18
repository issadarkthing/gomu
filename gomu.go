// Copyright (C) 2020  Raziman

package main

import (
	"log"
	"os"
	"path"

	"github.com/rivo/tview"
)

func main() {

	os.Setenv("TEST", "false")

	args := getArgs()

	readConfig(args)

	app := tview.NewApplication()

	start(app, args)

}

func init() {
	tmpDir := os.TempDir()

	logFile := path.Join(tmpDir, "gomu.log")

	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		log.Fatalf("Error opening file %s", logFile)
	}

	defer file.Close()

	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
}



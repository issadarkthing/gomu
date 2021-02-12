// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"

	"github.com/mattn/anko/vm"
)

// Panel is used to keep track of childrens in slices
type Panel interface {
	HasFocus() bool
	SetBorderColor(color tcell.Color) *tview.Box
	SetTitleColor(color tcell.Color) *tview.Box
	SetTitle(s string) *tview.Box
	GetTitle() string
	help() []string
}

const (
	configPath  = ".config/gomu/config"
	musicPath   = "~/music"
)

func execConfig() error {

	gomu.env.Define("echo", func(text string) {
		defaultTimedPopup(" Debug ", text)
	})

	content, err := ioutil.ReadFile("/home/terra/.config/gomu/config.anko")
	if err != nil {
		return tracerr.Wrap(err)
	}

	_, err = vm.Execute(gomu.env, nil, string(content))
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}



type Args struct {
	config  *string
	empty   *bool
	music   *string
	version *bool
}

func getArgs() Args {
	ar := Args{
		config:  flag.String("config", configPath, "Specify config file"),
		empty:   flag.Bool("empty", false, "Open gomu with empty queue. Does not override previous queue"),
		music:   flag.String("music", musicPath, "Specify music directory"),
		version: flag.Bool("version", false, "Print gomu version"),
	}
	flag.Parse()
	return ar
}

// Sets the layout of the application
func layout(gomu *Gomu) *tview.Flex {
	flex := tview.NewFlex().
		AddItem(gomu.playlist, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(gomu.queue, 0, 5, false).
			AddItem(gomu.playingBar, 0, 1, false), 0, 3, false)

	return flex
}

// Initialize
func start(application *tview.Application, args Args) {

	// Print version and exit
	if *args.version {
		fmt.Printf("Gomu %s\n", VERSION)
		return
	}

	// Assigning to global variable gomu
	gomu = newGomu()
	err := execConfig()
	if err != nil {
		panic(err)
	}

	gomu.args = args
	gomu.colors = newColor()

	// override default border
	// change double line border to one line border when focused
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
	tview.Styles.PrimitiveBackgroundColor = gomu.colors.background

	gomu.initPanels(application, args)
	gomu.command.defineCommands()

	flex := layout(gomu)
	gomu.pages.AddPage("main", flex, true, true)

	// sets the first focused panel
	gomu.setFocusPanel(gomu.playlist)
	gomu.prevPanel = gomu.playlist

	gomu.playingBar.setDefault()

	isQueueLoop, err := getBool(gomu.env, "queue_loop")
	if err != nil {
		logError(err)
		return
	}

	gomu.player.isLoop = isQueueLoop
	gomu.queue.isLoop = gomu.player.isLoop

	loadQueue, err := getBool(gomu.env, "load_prev_queue")
	if err != nil {
		logError(err)
		return
	}

	if !*args.empty && loadQueue {
		// load saved queue from previous session
		if err := gomu.queue.loadQueue(); err != nil {
			logError(err)
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		errMsg := fmt.Sprintf("Received %s. Exiting program", sig.String())
		logError(errors.New(errMsg))
		err := gomu.quit(args)
		if err != nil {
			logError(errors.New("unable to quit program"))
		}
	}()

	// global keybindings are handled here
	application.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		popupName, _ := gomu.pages.GetFrontPage()

		// disables keybindings when writing in input fields
		if strings.Contains(popupName, "-input-") {
			return e
		}

		switch e.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			if strings.Contains(popupName, "confirmation-") {
				return e
			}
			gomu.cyclePanels2()
		}

		cmds := map[rune]string{
			'q': "quit",
			' ': "toggle_pause",
			'+': "volume_up",
			'=': "volume_up",
			'-': "volume_down",
			'_': "volume_down",
			'n': "skip",
			':': "command_search",
			'?': "toggle_help",
			'f': "forward",
			'F': "forward_fast",
			'b': "rewind",
			'B': "rewind_fast",
		}

		for key, cmd := range cmds {
			if e.Rune() != key {
				continue
			}
			fn, err := gomu.command.getFn(cmd)
			if err != nil {
				logError(err)
				return e
			}
			fn()
		}

		return e
	})

	// fix transparent background issue
	application.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	go populateAudioLength(gomu.playlist.GetRoot())
	// main loop
	if err := application.SetRoot(gomu.pages, true).SetFocus(gomu.playlist).Run(); err != nil {
		logError(err)
	}

}

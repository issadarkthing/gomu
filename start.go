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

// Default values for command line arguments.
const (
	configPath = "~/.config/gomu/config"
	musicPath  = "~/music"
)



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


// executes user config with default config is executed first in order to apply
// default values
func execConfig(config string) error {

	const defaultConfig = `

// confirmation popup to add the whole playlist to the queue
confirm_bulk_add    = true
confirm_on_exit     = true
queue_loop          = false
load_prev_queue     = true
popup_timeout       = "5s"
// change this to directory that contains mp3 files
music_dir           = "~/music"
// url history of downloaded audio will be saved here
history_path        = "~/.local/share/gomu/urls"
// some of the terminal supports unicode character
// you can set this to true to enable emojis
use_emoji           = true
// initial volume when gomu starts up
volume              = 80
// if you experiencing error using this invidious instance, you can change it
// to another instance from this list:
// https://github.com/iv-org/documentation/blob/master/Invidious-Instances.md
invidious_instance  = "https://vid.puffyan.us"

// default emoji here is using awesome-terminal-fonts
// you can change these to your liking
emoji_playlist     = ""
emoji_file         = ""
emoji_loop         = "ﯩ"
emoji_noloop       = ""

// not all colors can be reproducible in terminal
// changing hex colors may or may not produce expected result
color_accent            = "#008B8B"
color_background        = "none"
color_foreground        = "#FFFFFF"
color_now_playing_title = "#017702"
color_playlist          = "#008B8B"
color_popup             = "#0A0F14"

// vim: syntax=go
`

	// built-in functions
	gomu.anko.Define("debug_popup", debugPopup)
	gomu.anko.Define("input_popup", inputPopup)
	gomu.anko.Define("show_popup", defaultTimedPopup)
	gomu.anko.Define("shell", shell)

	cfg := expandTilde(config)

	_, err := os.Stat(cfg)
	if os.IsNotExist(err) {
		err = appendFile(cfg, defaultConfig)
		if err != nil {
			return tracerr.Wrap(err)
		}
	}

	content, err := ioutil.ReadFile(cfg)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// execute default config
	_, err = gomu.anko.Execute(defaultConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// execute user config
	_, err = gomu.anko.Execute(string(content))
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
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
	err := execConfig(expandFilePath(*args.config))
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

	isQueueLoop := gomu.anko.GetBool("queue_loop")

	gomu.player.isLoop = isQueueLoop
	gomu.queue.isLoop = gomu.player.isLoop

	loadQueue := gomu.anko.GetBool("load_prev_queue")

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

		if gomu.anko.KeybindExists("global", string(e.Rune())) {
			// check for user defined keybindings
			gomu.anko.ExecKeybind("global", string(e.Rune()), func (err error) {
				if err != nil {
					errorPopup(tracerr.Wrap(err))
				}
			})

			return e
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

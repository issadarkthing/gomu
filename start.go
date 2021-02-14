// Copyright (C) 2020  Raziman

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"reflect"
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
	configPath = "~/.config/gomu/config"
	musicPath  = "~/music"
)

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
	gomu.env.DefineGlobal("debug_popup", debugPopup)
	gomu.env.DefineGlobal("input_popup", inputPopup)
	gomu.env.DefineGlobal("show_popup", defaultTimedPopup)

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
	_, err = vm.Execute(gomu.env, nil, defaultConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// execute user config
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

	isQueueLoop := getBool(gomu.env, "queue_loop")

	gomu.player.isLoop = isQueueLoop
	gomu.queue.isLoop = gomu.player.isLoop

	loadQueue := getBool(gomu.env, "load_prev_queue")

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

		// check for user defined keybindings
		kb, err := gomu.env.Get("keybinds")
		if err == nil {
			keybinds, ok := kb.(map[interface{}]interface{})
			if !ok {
				errorPopup(errors.New("invalid type; require {}"))
				return e
			}

			cmd, ok := keybinds[string(e.Rune())]
			if ok {

				f, ok := cmd.(func(context.Context) (reflect.Value, reflect.Value))
				if !ok {
					errorPopup(errors.New("invalid type; require type func()"))
					return e
				}

				go func() {
					_, execErr := f(context.Background())
					if err := execErr.Interface(); !execErr.IsNil() {
						if err, ok := err.(error); ok {
							errorPopup(err)
						}
					}
				}()

				return e
			}
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

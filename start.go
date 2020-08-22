// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

// Created so we can keep track of childrens in slices
type Panel interface {
	HasFocus() bool
	SetBorderColor(color tcell.Color) *tview.Box
	SetTitleColor(color tcell.Color) *tview.Box
	SetTitle(s string) *tview.Box
	GetTitle() string
	help() []string
}

const (
	CONFIG_PATH = ".config/gomu/config"
	MUSIC_PATH  = "~/music"
)

// Reads config file and sets the options
func readConfig(args Args) {

	const config = `
color:
  accent:            "#008B8B"
  background:        none
  foreground:        "#FFFFFF"
  now_playing_title: "#017702"
  playlist:          "#008B8B"
  popup:             "#0A0F14"

general:
  confirm_bulk_add:  true
  confirm_on_exit:   true
  load_prev_queue:   true
  music_dir:         ~/music
  popup_timeout:     5s
  volume:            100
  emoji:             true
  fzf:               false
`

	// config path passed by flag
	configPath := *args.config
	home, err := os.UserHomeDir()

	if err != nil {
		logError(err)
	}

	defaultPath := path.Join(home, CONFIG_PATH)

	if err != nil {
		logError(err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(strings.TrimSuffix(expandFilePath(configPath), "/config"))
	viper.AddConfigPath("$HOME/.config/gomu")

	if err := viper.ReadInConfig(); err != nil {

		// General config
		viper.SetDefault("general.music_dir", MUSIC_PATH)
		viper.SetDefault("general.confirm_on_exit", true)
		viper.SetDefault("general.confirm_bulk_add", true)
		viper.SetDefault("general.popup_timeout", "5s")
		viper.SetDefault("general.volume", 100)
		viper.SetDefault("general.load_prev_queue", true)
		viper.SetDefault("general.use_emoji", true)

		// creates gomu config dir if does not exist
		if _, err := os.Stat(defaultPath); err != nil {
			if err := os.MkdirAll(home+"/.config/gomu", 0755); err != nil {
				logError(err)
			}
		}

		// if config file was not found
		// copy default config to default config path
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {

			err = ioutil.WriteFile(defaultPath, []byte(config), 0644)
			if err != nil {
				logError(err)
			}

		}

	}
}

type Args struct {
	config  *string
	empty   *bool
	music   *string
	version *bool
}

func getArgs() Args {
	ar := Args{
		config:  flag.String("config", CONFIG_PATH, "Specify config file"),
		empty:   flag.Bool("empty", false, "Open gomu with empty queue. Does not override previous queue"),
		music:   flag.String("music", MUSIC_PATH, "Specify music directory"),
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

	flex := layout(gomu)
	gomu.pages.AddPage("main", flex, true, true)

	// sets the first focused panel
	gomu.setFocusPanel(gomu.playlist)
	gomu.prevPanel = gomu.playlist

	if !*args.empty && viper.GetBool("general.load_prev_queue") {
		// load saved queue from previous session
		if err := gomu.queue.loadQueue(); err != nil {
			logError(err)
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGKILL)
	go func() {
		sig := <-sigs
		errMsg := fmt.Sprintf("Received %s. Exiting program", sig.String())
		logError(errors.New(errMsg))
		err := gomu.quit(args)
		if err != nil {
			logError(errors.New("Unable to quit program"))
		}
	}()

	// global keybindings are handled here
	application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		popupName, _ := gomu.pages.GetFrontPage()

		// disables keybindings when writing in input fields
		if strings.Contains(popupName, "-input-") {
			return event
		}

		switch event.Key() {
		// cycle through each section
		case tcell.KeyTAB:

			if strings.Contains(popupName, "confirmation-") {
				return event
			}

			gomu.cyclePanels()

		}

		switch event.Rune() {
		case 'q':

			if !viper.GetBool("general.confirm_on_exit") {
				err := gomu.quit(args)
				if err != nil {
					logError(err)
				}
			}

			exitConfirmation(args)

		case ' ':
			gomu.player.togglePause()

		case '+':
			v := int(gomu.player.volume*10) + 100
			if v < 100 {
				vol := gomu.player.setVolume(0.5)
				volumePopup(vol)
			}

		case '-':
			v := int(gomu.player.volume*10) + 100
			if v > 0 {
				vol := gomu.player.setVolume(-0.5)
				volumePopup(vol)
			}

		case 'n':
			gomu.player.skip()

		case '?':

			name, _ := gomu.pages.GetFrontPage()

			if name == "help-page" {
				gomu.pages.RemovePage(name)
				gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))
			} else {
				helpPopup(gomu.prevPanel)
			}

		}

		return event
	})

	// fix transparent background issue
	application.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	// main loop
	if err := application.SetRoot(gomu.pages, true).SetFocus(gomu.playlist).Run(); err != nil {
		logError(err)
	}
}

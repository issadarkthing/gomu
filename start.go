// Copyright (C) 2020  Raziman

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

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

func readConfig(args Args) {

	configPath := *args.config
	musicDir := *args.music
	home, err := os.UserHomeDir()

	if err != nil {
		logError(err)
	}

	defaultPath := path.Join(home, ".config/gomu/config")

	if err != nil {
		logError(err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(expandTilde(configPath))
	viper.AddConfigPath("$HOME/.config/gomu")

	colors := map[string]string{
		"color.foreground":        "#FFFFFF",
		"color.background":        "none",
		"color.accent":            "#008B8B",
		"color.popup":             "#0A0F14",
		"color.now_playing_title": "#017702",
		"color.playlist":          "#008B8B",
	}

	if err := viper.ReadInConfig(); err != nil {

		// General config
		viper.SetDefault("general.music_dir", musicDir)
		viper.SetDefault("general.confirm_on_exit", true)
		viper.SetDefault("general.confirm_bulk_add", true)
		viper.SetDefault("general.popup_timeout", "5s")
		viper.SetDefault("general.volume", 100)
		viper.SetDefault("general.load_prev_queue", true)

		// Colors
		for k, v := range colors {
			viper.SetDefault(k, v)
		}

		// creates gomu config dir if does not exist
		if _, err := os.Stat(defaultPath); err != nil {
			if err := os.MkdirAll(home+"/.config/gomu", 0755); err != nil {
				logError(err)
			}
		}

		// if config file was not found
		// copy default config to default config path
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {

			input, err := ioutil.ReadFile("config")
			if err != nil {
				logError(err)
			}

			err = ioutil.WriteFile(defaultPath, input, 0644)
			if err != nil {
				logError(err)
			}

		}

	} else {

		// Validate hex color
		for k, v := range colors {
			cfgColor := viper.GetString(k)
			if validateHexColor(cfgColor) {
				continue
			}
			// use default value if invalid hex color was given
			viper.Set(k, v)
		}
	}

}

type Args struct {
	config  *string
	load    *bool
	music   *string
	version *bool
}

func getArgs() Args {
	ar := Args{
		config:  flag.String("config", "~/.config/gomu/config", "specify config file"),
		load:    flag.Bool("load", true, "load previous queue"),
		music:   flag.String("music", "~/music", "specify music directory"),
		version: flag.Bool("version", false, "print gomu version"),
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

	// override default border
	// change double line border to one line border when focused
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight

	var bgColor tcell.Color
	bg := viper.GetString("color.background")
	if bg == "none" {
		bgColor = tcell.ColorDefault
	} else {
		bgColor = tcell.GetColor(bg)
	}

	tview.Styles.PrimitiveBackgroundColor = bgColor

	// Assigning to global variable gomu
	gomu = newGomu()
	gomu.initPanels(application)

	debugLog("App start")

	flex := layout(gomu)
	gomu.pages.AddPage("main", flex, true, true)

	// sets the first focused panel
	gomu.setFocusPanel(gomu.playlist)
	gomu.prevPanel = gomu.playlist

	if *args.load {
		// load saved queue from previous
		if err := gomu.queue.loadQueue(); err != nil {
			logError(err)
		}
	}

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
				err := gomu.quit()
				if err != nil {
					logError(err)
				}
			}

			exitConfirmation()

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

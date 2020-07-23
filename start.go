// Copyright (C) 2020  Raziman

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

// created so we can keep track of childrens in slices
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

	defaultPath := home + "/.config/gomu/config"

	if err != nil {
		logError(err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(expandTilde(configPath))
	viper.AddConfigPath("/etc/gomu")
	viper.AddConfigPath("$HOME/.gomu")
	viper.AddConfigPath("$HOME/.config/gomu")

	if err := viper.ReadInConfig(); err != nil {

		viper.SetDefault("music_dir", musicDir)
		viper.SetDefault("confirm_on_exit", true)
		viper.SetDefault("confirm_bulk_add", true)
		viper.SetDefault("popup_timeout", "5s")
		viper.SetDefault("volume", "50")

		// creates gomu config dir if does not exist
		if _, err := os.Stat(defaultPath); err != nil {
			if err := os.MkdirAll(home+"/.config/gomu", 0755); err != nil {
				logError(err)
			}
		}

		// if config file was not found
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := viper.SafeWriteConfigAs(defaultPath); err != nil {
				logError(err)
			}
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
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.BorderColor = tcell.ColorWhite

	// Assigning to global variable gomu
	gomu = newGomu()
	gomu.initPanels(application)

	logError(fmt.Errorf("App start"))

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

		switch event.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			gomu.cyclePanels()

		}

		name, _ := gomu.pages.GetFrontPage()

		// disables keybindings when writing in input fields
		if strings.Contains(name, "-input-") {
			return event
		}

		switch event.Rune() {
		case 'q':

			if !viper.GetBool("confirm_on_exit") {
				application.Stop()
			}
			confirmationPopup("Are you sure to exit?", func(_ int, label string) {

				if label == "no" || label == "" {
					gomu.pages.RemovePage("confirmation-popup")
					return
				}

				if err := gomu.queue.saveQueue(); err != nil {
					logError(err)
				}

				if err := viper.WriteConfig(); err != nil {
					logError(err)
				}

				application.Stop()

			})

		case ' ':
			gomu.player.togglePause()

		case '+':
			v := int(gomu.player.volume*10) + 50
			if v < 50 {
				vol := gomu.player.setVolume(0.5)
				volumePopup(vol)
			}

		case '-':
			v := int(gomu.player.volume*10) + 50
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

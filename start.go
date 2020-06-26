// Copyright (C) 2020  Raziman

package main

import (
	"os"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

var (
	app        *tview.Application
	playingBar *PlayingBar
	queue      *Queue
	playlist   *Playlist
	player     *Player
	pages      *tview.Pages
)

func start(application *tview.Application) {
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

	player = &Player{}
	app = application
	playingBar = InitPlayingBar()
	queue = InitQueue()
	playlist = InitPlaylist()

	flex := Layout()
	pages = tview.NewPages().AddPage("main", flex, true, true)

	childrens := []Children{playlist, queue, playingBar}



	application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		switch event.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			cycleChildren(application, childrens)

		}

		switch event.Rune() {
		case 'q':

			if !viper.GetBool("confirm_on_exit") {
				application.Stop()
			}

			confirmationPopup("Are you sure to exit?", func(_ int, label string) {

				if label == "yes" {
					application.Stop()
				} else {
					pages.RemovePage("confirmation-popup")
				}

			})

		case ' ':
			player.TogglePause()

		case '+':
			player.Volume(0.5)

		case '-':
			player.Volume(-0.5)

		case 'n':
			player.Skip()

		}

		return event
	})

	// fix transparent background issue
	application.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	// main loop
	if err := application.SetRoot(pages, true).SetFocus(flex).Run(); err != nil {
		log(err.Error())
	}
}

// created so we can keep track of childrens in slices
type Children interface {
	HasFocus() bool
	SetBorderColor(color tcell.Color) *tview.Box
	SetTitleColor(color tcell.Color) *tview.Box
	SetTitle(s string) *tview.Box
	GetTitle() string
}

func cycleChildren(app *tview.Application, childrens []Children) Children {

	focusedColor := accentColor
	unfocusedColor := textColor
	anyChildHasFocus := false

	for i, child := range childrens {

		if child.HasFocus() {

			anyChildHasFocus = true

			var nextChild Children

			// if its the last element set the child back to one
			if i == len(childrens)-1 {
				nextChild = childrens[0]
			} else {
				nextChild = childrens[i+1]
			}

			child.SetBorderColor(unfocusedColor)
			child.SetTitleColor(unfocusedColor)

			app.SetFocus(nextChild.(tview.Primitive))
			nextChild.SetBorderColor(focusedColor)
			nextChild.SetTitleColor(focusedColor)

			return nextChild
		}
	}
first := childrens[0]

	if anyChildHasFocus == false {

		app.SetFocus(first.(tview.Primitive))
		first.SetBorderColor(focusedColor)
		first.SetTitleColor(focusedColor)

	}

	return first
}

func readConfig() {

	home, err := os.UserHomeDir()
	configPath := home + "/.config/gomu/config"

	if err != nil {
		panic(err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/gomu")
	viper.AddConfigPath(home + "/.gomu")
	viper.AddConfigPath("$HOME/.config/gomu")

	if err := viper.ReadInConfig(); err != nil {

		viper.SetDefault("music_dir", "~/music")
		viper.SetDefault("confirm_on_exit", true)

		// creates gomu config dir if does not exist
		if _, err := os.Stat(configPath); err != nil {
			if err := os.MkdirAll(home+"/.config/gomu", 0755); err != nil {
				panic(err)
			}
		}

		// if config file was not found
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := viper.SafeWriteConfigAs(configPath); err != nil {
				panic(err)
			}
		}

	}

}

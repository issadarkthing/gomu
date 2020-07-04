// Copyright (C) 2020  Raziman

package main

import (
	"os"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

// var (
// 	app         *tview.Application
// 	playingBar  *PlayingBar
// 	queue       *Queue
// 	playlist    *Playlist
// 	player      *Player
// 	pages       *tview.Pages
// 	prevPanel   Children
// 	popupBg     = tcell.GetColor("#0A0F14")
// 	textColor   = tcell.ColorWhite
// 	accentColor = tcell.ColorDarkCyan
// )

type Gomu struct {
	App         *tview.Application
	PlayingBar  *PlayingBar
	Queue       *Queue
	Playlist    *Playlist
	Player      *Player
	Pages       *tview.Pages
	PrevPanel   Children
	PopupBg     tcell.Color
	TextColor   tcell.Color
	AccentColor tcell.Color
}

// Creates new instance of gomu with default values
func NewGomu() *Gomu {

	gomu := &Gomu{
		PopupBg:     tcell.GetColor("#0A0F14"),
		TextColor:   tcell.ColorWhite,
		AccentColor: tcell.ColorDarkCyan,
	}

	return gomu
}

// Initialize childrens/panels this is seperated from 
// constructor function `NewGomu` so that we can 
// test independently
func (g *Gomu) InitChildrens(app *tview.Application) {
	g.App = app
	g.PlayingBar = NewPlayingBar()
	g.Queue = NewQueue()
	g.Playlist = NewPlaylist()
	g.Player = &Player{}
	g.Pages = tview.NewPages()
}

var gomu *Gomu

func start(application *tview.Application) {
	// override default border
	// change double line border to one line border when focused
	tview.Borders.HorizontalFocus         = tview.Borders.Horizontal
	tview.Borders.VerticalFocus           = tview.Borders.Vertical
	tview.Borders.TopLeftFocus            = tview.Borders.TopLeft
	tview.Borders.TopRightFocus           = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus         = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus        = tview.Borders.BottomRight
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.BorderColor              = tcell.ColorWhite

	gomu = NewGomu()
	gomu.InitChildrens(application)


	appLog("start app")

	flex := Layout(gomu)
	gomu.Pages.AddPage("main", flex, true, true)

	gomu.Playlist.SetBorderColor(gomu.AccentColor)
	gomu.Playlist.SetTitleColor(gomu.AccentColor)
	gomu.PrevPanel = gomu.Playlist

	childrens := []Children{gomu.Playlist, gomu.Queue, gomu.PlayingBar}

	application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		switch event.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			gomu.PrevPanel = cycleChildren(gomu, childrens) 
		}

		name, _ := gomu.Pages.GetFrontPage()

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

				if label == "yes" {
					application.Stop()
				} else {
					gomu.Pages.RemovePage("confirmation-popup")
				}

			})

		case ' ':
			gomu.Player.TogglePause()

		case '+':
			v := int(gomu.Player.volume*10) + 50
			if v < 50 {
				vol := gomu.Player.Volume(0.5)
				volumePopup(vol)
			}

		case '-':
			v := int(gomu.Player.volume*10) + 50
			if v > 0 {
				vol := gomu.Player.Volume(-0.5)
				volumePopup(vol)
			}

		case 'n':
			gomu.Player.Skip()

		case '?':

			name, _ := gomu.Pages.GetFrontPage()

			if name == "help-page" {
				gomu.Pages.RemovePage(name)
				gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
			} else {
				helpPopup()
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
	if err := application.SetRoot(gomu.Pages, true).SetFocus(gomu.Playlist).Run(); err != nil {
		appLog(err)
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

func cycleChildren(gomu *Gomu, childrens []Children) Children {

	focusedColor := gomu.AccentColor
	unfocusedColor := gomu.TextColor
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

			gomu.App.SetFocus(nextChild.(tview.Primitive))
			nextChild.SetBorderColor(focusedColor)
			nextChild.SetTitleColor(focusedColor)

			return nextChild
		}
	}

	first := childrens[0]

	if !anyChildHasFocus {

		gomu.App.SetFocus(first.(tview.Primitive))
		first.SetBorderColor(focusedColor)
		first.SetTitleColor(focusedColor)

	}

	return first
}

func readConfig() {

	home, err := os.UserHomeDir()
	configPath := home + "/.config/gomu/config"

	if err != nil {
		appLog(err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/gomu")
	viper.AddConfigPath(home + "/.gomu")
	viper.AddConfigPath("$HOME/.config/gomu")

	if err := viper.ReadInConfig(); err != nil {

		viper.SetDefault("music_dir", "~/music")
		viper.SetDefault("confirm_on_exit", true)
		viper.SetDefault("confirm_bulk_add", true)
		viper.SetDefault("popup_timeout", "5s")

		// creates gomu config dir if does not exist
		if _, err := os.Stat(configPath); err != nil {
			if err := os.MkdirAll(home+"/.config/gomu", 0755); err != nil {
				appLog(err)
			}
		}

		// if config file was not found
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := viper.SafeWriteConfigAs(configPath); err != nil {
				appLog(err)
			}
		}

	}

}

// layout is used to organize the panels
func Layout(gomu *Gomu) *tview.Flex {

	flex := tview.NewFlex().
		AddItem(gomu.Playlist, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(gomu.Queue, 0, 7, false).
			AddItem(gomu.PlayingBar, 0, 1, false), 0, 3, false)

	return flex

}

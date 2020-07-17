// Copyright (C) 2020  Raziman

package main

import (
	"log"
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
	Help() []string
}

type Gomu struct {
	App         *tview.Application
	PlayingBar  *PlayingBar
	Queue       *Queue
	Playlist    *Playlist
	Player      *Player
	Pages       *tview.Pages
	Popups      []tview.Primitive
	PrevPanel   Panel
	PopupBg     tcell.Color
	TextColor   tcell.Color
	AccentColor tcell.Color
	Panels      []Panel
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
func (g *Gomu) InitPanels(app *tview.Application) {
	g.App = app
	g.PlayingBar = NewPlayingBar()
	g.Queue = NewQueue()
	g.Playlist = NewPlaylist()
	g.Player = NewPlayer()
	g.Pages = tview.NewPages()
	g.Panels = []Panel{g.Playlist, g.Queue, g.PlayingBar}
}

// cycle between panels
func (g *Gomu) CyclePanels() Panel {

	var anyChildHasFocus bool

	for i, child := range g.Panels {

		if child.HasFocus() {

			anyChildHasFocus = true

			var nextChild Panel

			// if its the last element set the child back to one
			if i == len(g.Panels)-1 {
				nextChild = g.Panels[0]
			} else {
				nextChild = g.Panels[i+1]
			}

			g.SetFocusPanel(nextChild)

			g.PrevPanel = nextChild
			return nextChild
		}
	}

	first := g.Panels[0]

	if !anyChildHasFocus {
		g.SetFocusPanel(first)
	}

	g.PrevPanel = first
	return first
}

// changes title and border color when focusing panel
// and changes color of the previous panel as well
func (g *Gomu) SetFocusPanel(panel Panel) {

	g.App.SetFocus(panel.(tview.Primitive))
	panel.SetBorderColor(g.AccentColor)
	panel.SetTitleColor(g.AccentColor)

	if g.PrevPanel == nil {
		return
	}

	g.SetUnfocusPanel(g.PrevPanel)
}

// removes the color of the given panel
func (g *Gomu) SetUnfocusPanel(panel Panel) {
	g.PrevPanel.SetBorderColor(g.TextColor)
	g.PrevPanel.SetTitleColor((g.TextColor))
}

// one single instance of global variable
var gomu *Gomu

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

	gomu = NewGomu()
	gomu.InitPanels(application)

	log.Println("start app")

	flex := Layout(gomu)
	gomu.Pages.AddPage("main", flex, true, true)

	gomu.Playlist.SetBorderColor(gomu.AccentColor)
	gomu.Playlist.SetTitleColor(gomu.AccentColor)
	gomu.PrevPanel = gomu.Playlist

	if err := gomu.Queue.LoadQueue(); err != nil {
		log.Println(err)
	}

	application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		switch event.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			gomu.CyclePanels()

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

				if label == "no" {
					gomu.Pages.RemovePage("confirmation-popup")
					return
				}

				if err := gomu.Queue.SaveQueue(); err != nil {
					log.Println(err)
				}

				if err := viper.WriteConfig(); err != nil {
					log.Println(err)
				}

				application.Stop()

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
				helpPopup(gomu.PrevPanel)
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
		log.Println(err)
	}
}

func readConfig() {

	home, err := os.UserHomeDir()
	configPath := home + "/.config/gomu/config"

	if err != nil {
		log.Println(err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/gomu")
	viper.AddConfigPath("$HOME/.gomu")
	viper.AddConfigPath("$HOME/.config/gomu")

	if err := viper.ReadInConfig(); err != nil {

		viper.SetDefault("music_dir", "~/music")
		viper.SetDefault("confirm_on_exit", true)
		viper.SetDefault("confirm_bulk_add", true)
		viper.SetDefault("popup_timeout", "5s")
		viper.SetDefault("volume", "50")

		// creates gomu config dir if does not exist
		if _, err := os.Stat(configPath); err != nil {
			if err := os.MkdirAll(home+"/.config/gomu", 0755); err != nil {
				log.Println(err)
			}
		}

		// if config file was not found
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := viper.SafeWriteConfigAs(configPath); err != nil {
				log.Println(err)
			}
		}

	}

}

// layout is used to organize the panels
func Layout(gomu *Gomu) *tview.Flex {

	flex := tview.NewFlex().
		AddItem(gomu.Playlist, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(gomu.Queue, 0, 5, false).
			AddItem(gomu.PlayingBar, 0, 1, false), 0, 3, false)

	return flex

}

// Copyright (C) 2020  Raziman

package main

import (
	"os"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

func start(app *tview.Application) {
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

	player := &Player{}


	playingBar := PlayingBar(app, player)
	queue := Queue(player)
	playlist := Playlist(player)

	player.tree = playlist
	player.list = queue
	player.playingBar = playingBar
	player.app = app

	flex := Layout(app, player)
	pages := tview.NewPages().AddPage("main", flex, true, true)

	playlist.SetInputCapture(func (e *tcell.EventKey) *tcell.EventKey {


		currNode := playlist.GetCurrentNode()

		if currNode == playlist.GetRoot() {
			return e
		}

		audioFile := currNode.GetReference().(*AudioFile)

		switch e.Rune() {
		case 'l':

			log("test")
			addToQueue(audioFile, player, queue)
			currNode.SetExpanded(true)

		case 'h':

			// if closing node with no children
			// close the node's parent
			// remove the color of the node

			if audioFile.IsAudioFile {
				parent := audioFile.Parent

				currNode.SetColor(textColor)
				parent.SetExpanded(false)
				parent.SetColor(accentColor)
				//prevNode = parent
				playlist.SetCurrentNode(parent)
			}

			currNode.Collapse()

		case 'L':

			confirmationPopup(
				app, 
				pages, 
				"Are you sure to add this whole directory into queue?", 
				func (_ int, label string) {

					if label == "yes" {
						addAllToQueue(playlist.GetCurrentNode(), player, queue)
					} 					

					pages.RemovePage("confirmation-popup")

				})

		}


		return e
	})

	childrens := []Children{playlist, queue, playingBar.frame}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		switch event.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			cycleChildren(app, childrens)

		}

		switch event.Rune() {
		case 'q':

			if !viper.GetBool("confirm_on_exit") {
				app.Stop()
			}


			confirmationPopup(app, pages, "Are you sure to exit?", func(_ int, label string) {

				if label == "yes" {
					app.Stop()
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
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	// main loop
	if err := app.SetRoot(pages, true).SetFocus(flex).Run(); err != nil {
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

func cycleChildren(app *tview.Application, childrens []Children) {

	focusedColor := tcell.ColorDarkCyan
	unfocusedColor := tcell.ColorAntiqueWhite
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

			break
		}
	}

	if anyChildHasFocus == false {

		app.SetFocus(childrens[0].(tview.Primitive))
		childrens[0].SetBorderColor(focusedColor)
		childrens[0].SetTitleColor(focusedColor)
	}

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
			if err := os.MkdirAll(home + "/.config/gomu", 0755); err != nil {
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

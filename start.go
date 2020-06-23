// Copyright (C) 2020  Raziman

package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
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
	tview.Styles.BorderColor = tcell.ColorAntiqueWhite

	player := &Player{}

	child3 := PlayingBar(app, player)
	child2 := Queue(player)
	child1 := Playlist(child2, child3, player)

	player.tree = child1
	player.list = child2
	player.playingBar = child3
	player.app = app

	flex := Layout(app, player)

	pages := tview.NewPages().AddPage("main", flex, true, true)

	childrens := []Children{child1, child2, child3.frame}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		switch event.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			cycleChildren(app, childrens)

		}

		switch event.Rune() {
		case 'q':

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

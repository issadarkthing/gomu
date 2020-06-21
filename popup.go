// Copyright (C) 2020  Raziman

package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func confirmationPopup(
	app *tview.Application,
	pages *tview.Pages,
	text string,
	handler func(buttonIndex int, buttonLabel string),
) {

	modal := tview.NewModal().
		SetText(text).
		SetBackgroundColor(tcell.ColorDefault).
		AddButtons([]string{"yes", "no"}).
		SetButtonBackgroundColor(tcell.ColorBlack).
		SetDoneFunc(handler)

	pages.AddPage("confirmation-popup", center(modal, 40, 10), true, true)
	app.SetFocus(modal)

}

func center(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, false).
			AddItem(nil, 0, 1, false), width, 1, false).
		AddItem(nil, 0, 1, false)
}

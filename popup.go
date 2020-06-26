// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"
)

func confirmationPopup(
	text string,
	handler func(buttonIndex int, buttonLabel string),
) {

	modal := tview.NewModal().
		SetText(text).
		SetBackgroundColor(popupBg).
		AddButtons([]string{"yes", "no"}).
		SetButtonBackgroundColor(popupBg).
		SetButtonTextColor(accentColor).
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

func topRight(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 23, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, false).
			AddItem(nil, 0, 15, false), width, 1, false).
		AddItem(nil, 0, 1, false)
}

func timeoutPopup(title string, desc string, timeout time.Duration) {


	textView := tview.NewTextView().
		SetText(fmt.Sprintf("%s", desc)).
		SetTextColor(accentColor)

	textView.SetTextAlign(tview.AlignCenter).SetBackgroundColor(popupBg)

	box := tview.NewFrame(textView).SetBorders(1, 1, 1, 1, 1, 1)
	box.SetTitle(title).SetBorder(true).SetBackgroundColor(popupBg)

	pages.AddPage("timeout-popup", topRight(box, 70, 7), true, true)
	app.SetFocus(prevPanel.(tview.Primitive))

	go func() {
		time.Sleep(timeout)
		pages.RemovePage("timeout-popup")
		app.SetFocus(prevPanel.(tview.Primitive))
	}()
}


func volumePopup(volume float64) {

	vol := int(volume * 10) + 50

	progress := fmt.Sprintf("\n%d |%s%s| %s",
		vol,
		strings.Repeat("█", vol),
		strings.Repeat("-", 50-vol),
		"50",
	)

	timeoutPopup(" Volume ", progress, time.Second * 5)

}

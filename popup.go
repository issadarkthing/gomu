// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// this is used to make the popup unique
// this mitigates the issue of closing all popups when timeout ends
var popupCounter = 0

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

	popupId := fmt.Sprintf("%s %d", "timeout-popup", popupCounter)

	pages.AddPage(popupId, topRight(box, 70, 7), true, true)
	app.SetFocus(prevPanel.(tview.Primitive))

	go func() {
		time.Sleep(timeout)
		pages.RemovePage(popupId)
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

func helpPopup() {

	helpText := []string{
		"j      down",
		"k      up",
		"tab    change panel",
		"space  toggle play/pause",
		"n      skip",
		"q      quit",
		"l      add song to queue",
		"L      add playlist to queue",
		"h      close node in playlist",
		"d      remove song from queue",
		"+      volume up",
		"-      volume down",
		"?      toggle help",
	}

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(popupBg).SetTitle(" Help ").
		 SetBorder(true)
	list.SetSelectedBackgroundColor(popupBg).
		 SetSelectedTextColor(accentColor)

	for _, v := range helpText {
		list.AddItem(v, "", 0, nil)
	}


	prev := func() {
		currIndex := list.GetCurrentItem()
		list.SetCurrentItem(currIndex - 1)
	}

	next := func() {
		currIndex := list.GetCurrentItem()
		idx := currIndex + 1
		if currIndex == list.GetItemCount()-1 {
			idx = 0
		}
		list.SetCurrentItem(idx)
	}

	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		switch e.Rune() {
		case 'j':
			next()
		case 'k':
			prev()
		case 'd':
			queue.deleteItem(queue.GetCurrentItem())
		}

		return nil
	})

	pages.AddPage("help-page", center(list, 50, 30), true, true)
	app.SetFocus(list)
}

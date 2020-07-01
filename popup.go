// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

// this is used to make the popup unique
// this mitigates the issue of closing all popups when timeout ends
var (
	popupCounter = 0
	popupTimeout = time.Duration(viper.GetInt("popup_timeout")) * time.Second
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

func timedPopup(title string, desc string, timeout time.Duration) {

	textView := tview.NewTextView().
		SetText(fmt.Sprintf("%s", desc)).
		SetTextColor(accentColor)

	textView.SetTextAlign(tview.AlignCenter).SetBackgroundColor(popupBg)

	box := tview.NewFrame(textView).SetBorders(1, 1, 1, 1, 1, 1)
	box.SetTitle(title).SetBorder(true).SetBackgroundColor(popupBg)

	popupId := fmt.Sprintf("%s %d", "timeout-popup", popupCounter)
	popupCounter++

	pages.AddPage(popupId, topRight(box, 70, 7), true, true)
	app.SetFocus(prevPanel.(tview.Primitive))

	go func() {
		time.Sleep(timeout)
		pages.RemovePage(popupId)
		app.SetFocus(prevPanel.(tview.Primitive))
	}()
}

func volumePopup(volume float64) {

	vol := int(volume*10) + 50

	progress := fmt.Sprintf("\n%d |%s%s| %s",
		vol,
		strings.Repeat("█", vol),
		strings.Repeat("-", 50-vol),
		"50",
	)

	timedPopup(" Volume ", progress, time.Second*5)

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
		"d      remove from queue",
		"+      volume up",
		"-      volume down",
		"?      toggle help",
		"Y      download audio",
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
		}
		
		switch e.Key() {
		case tcell.KeyEsc:
			pages.RemovePage("help-page")
			app.SetFocus(prevPanel.(tview.Primitive))
		}

		return nil
	})

	pages.AddPage("help-page", center(list, 50, 30), true, true)
	app.SetFocus(list)
}

func downloadMusic(selPlaylist *tview.TreeNode) {

	inputField := tview.NewInputField().
		SetLabel("Enter a url: ").
		SetFieldWidth(0).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50))

	inputField.SetBackgroundColor(popupBg).SetBorder(true).SetTitle(" Ytdl ")
	inputField.SetFieldBackgroundColor(accentColor).SetFieldTextColor(tcell.ColorBlack)

	inputField.SetDoneFunc(func(key tcell.Key) {

		switch key {
		case tcell.KeyEnter:
			url := inputField.GetText()
			Ytdl(url, selPlaylist)
			pages.RemovePage("download-input-popup")

		case tcell.KeyEscape:
			pages.RemovePage("download-input-popup")
		}

	})

	pages.AddPage("download-input-popup", center(inputField, 50, 4), true, true)
	app.SetFocus(inputField)

}

func CreatePlaylistPopup() {
	
	inputField := tview.NewInputField().
		SetLabel("Enter a playlist name: ").
		SetFieldWidth(0).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50))

	inputField.SetBackgroundColor(popupBg).SetBorder(true).SetTitle(" New Playlist ")
	inputField.SetFieldBackgroundColor(accentColor).SetFieldTextColor(tcell.ColorBlack)

	inputField.SetDoneFunc(func(key tcell.Key) {

		switch key {
		case tcell.KeyEnter:
			playListName := inputField.GetText()
			playlist.CreatePlaylist(playListName)
			pages.RemovePage("mkdir-input-popup")
			app.SetFocus(prevPanel.(tview.Primitive))

		case tcell.KeyEsc:
			pages.RemovePage("mkdir-input-popup")
		}

	})

	pages.AddPage("mkdir-input-popup", center(inputField, 50, 4), true, true)
	app.SetFocus(inputField)

}

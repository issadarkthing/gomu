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
)

// gets popup timeout from config file
func getPopupTimeout() time.Duration {

	dur := viper.GetString("popup_timeout")

	m, err := time.ParseDuration(dur)

	if err != nil {
		appLog(err)
	}

	return m

}

func confirmationPopup(
	text string,
	handler func(buttonIndex int, buttonLabel string),
) {

	modal := tview.NewModal().
		SetText(text).
		SetBackgroundColor(gomu.PopupBg).
		AddButtons([]string{"yes", "no"}).
		SetButtonBackgroundColor(gomu.PopupBg).
		SetButtonTextColor(gomu.AccentColor).
		SetDoneFunc(handler)

	gomu.Pages.AddPage("confirmation-popup", center(modal, 40, 10), true, true)
	gomu.App.SetFocus(modal)

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
		SetText(desc).
		SetTextColor(gomu.AccentColor)

	textView.SetTextAlign(tview.AlignCenter).SetBackgroundColor(gomu.PopupBg)

	box := tview.NewFrame(textView).SetBorders(1, 1, 1, 1, 1, 1)
	box.SetTitle(title).SetBorder(true).SetBackgroundColor(gomu.PopupBg)

	popupId := fmt.Sprintf("%s %d", "timeout-popup", popupCounter)
	popupCounter++

	gomu.Pages.AddPage(popupId, topRight(box, 70, 7), true, true)
	gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))

	go func() {
		time.Sleep(timeout)
		gomu.Pages.RemovePage(popupId)
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

	timedPopup(" Volume ", progress, getPopupTimeout())

}

func helpPopup() {

	helpText := []string{
		"j      down",
		"k      up",
		"tab    change panel",
		"space  toggle play/pause",
		"esc    close popup",
		"n      skip",
		"q      quit",
		"l      add song to queue",
		"L      add playlist to queue",
		"h      close node in playlist",
		"d      remove from queue",
		"D      delete playlist",
		"+      volume up",
		"-      volume down",
		"?      toggle help",
		"Y      download audio",
		"a      create playlist",
	}

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(gomu.PopupBg).SetTitle(" Help ").
		SetBorder(true)
	list.SetSelectedBackgroundColor(gomu.PopupBg).
		SetSelectedTextColor(gomu.AccentColor)

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
			gomu.Pages.RemovePage("help-page")
			gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
		}

		return nil
	})

	gomu.Pages.AddPage("help-page", center(list, 50, 30), true, true)
	gomu.App.SetFocus(list)
}

func downloadMusicPopup(selPlaylist *tview.TreeNode) {

	inputField := tview.NewInputField().
		SetLabel("Enter a url: ").
		SetFieldWidth(0).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50))

	inputField.SetBackgroundColor(gomu.PopupBg).SetBorder(true).SetTitle(" Ytdl ")
	inputField.SetFieldBackgroundColor(gomu.AccentColor).SetFieldTextColor(tcell.ColorBlack)

	inputField.SetDoneFunc(func(key tcell.Key) {

		switch key {
		case tcell.KeyEnter:
			url := inputField.GetText()
			Ytdl(url, selPlaylist)
			gomu.Pages.RemovePage("download-input-popup")

		case tcell.KeyEscape:
			gomu.Pages.RemovePage("download-input-popup")
		}

		gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))

	})

	gomu.Pages.AddPage("download-input-popup", center(inputField, 50, 4), true, true)
	gomu.App.SetFocus(inputField)

}

func CreatePlaylistPopup() {

	inputField := tview.NewInputField().
		SetLabel("Enter a playlist name: ").
		SetFieldWidth(0).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50))

	inputField.SetBackgroundColor(gomu.PopupBg).SetBorder(true).SetTitle(" New Playlist ")
	inputField.SetFieldBackgroundColor(gomu.AccentColor).SetFieldTextColor(tcell.ColorBlack)

	inputField.SetDoneFunc(func(key tcell.Key) {

		switch key {
		case tcell.KeyEnter:
			playListName := inputField.GetText()
			err := gomu.Playlist.CreatePlaylist(playListName)

			if err != nil {
				appLog(err)
			}

			gomu.Pages.RemovePage("mkdir-input-popup")
			gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))

		case tcell.KeyEsc:
			gomu.Pages.RemovePage("mkdir-input-popup")
			gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
		}

	})

	gomu.Pages.AddPage("mkdir-input-popup", center(inputField, 50, 4), true, true)
	gomu.App.SetFocus(inputField)

}

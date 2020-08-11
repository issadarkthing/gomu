// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
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

// Simple stack data structure
type Stack struct {
	popups []tview.Primitive
}

// Push popup to the stack and focus
func (s *Stack) push(p tview.Primitive) {
	s.popups = append(s.popups, p)
	gomu.app.SetFocus(p)
}

// Show item on the top of the stack
func (s *Stack) peekTop() tview.Primitive {

	if len(s.popups)-1 < 0 {
		return nil
	}

	return s.popups[len(s.popups)-1]
}

// Remove popup from the stack and focus previous popup
func (s *Stack) pop() tview.Primitive {

	if len(s.popups) == 0 {
		return nil
	}

	last := s.popups[len(s.popups)-1]
	res := s.popups[:len(s.popups)-1]
	s.popups = res

	// focus previous popup
	if len(s.popups) > 0 {
		gomu.app.SetFocus(s.popups[len(s.popups)-1])
	} else {
		// focus the panel if no popup left
		gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))
	}

	return last
}

// Gets popup timeout from config file
func getPopupTimeout() time.Duration {

	dur := viper.GetString("general.popup_timeout")
	m, err := time.ParseDuration(dur)

	if err != nil {
		logError(err)
		return time.Second * 5
	}

	return m
}

// Simple confirmation popup. Accepts callback
func confirmationPopup(
	text string,
	handler func(buttonIndex int, buttonLabel string),
) {

	modal := tview.NewModal().
		SetText(text).
		SetBackgroundColor(gomu.popupBg).
		AddButtons([]string{"no", "yes"}).
		SetButtonBackgroundColor(gomu.popupBg).
		SetButtonTextColor(gomu.accentColor).
		SetDoneFunc(func(indx int, label string) {
			handler(indx, label)
			gomu.pages.RemovePage("confirmation-popup")
			gomu.popups.pop()
		})

	gomu.pages.
		AddPage("confirmation-popup", center(modal, 40, 10), true, true)

	gomu.popups.push(modal)

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

// Width and height parameter is optional. It defaults to 70 and 7 respectively.
func timedPopup(
	title string, desc string, timeout time.Duration, width, height int,
) {

	// Wait until app is not suspended
	for {
		if !gomu.isSuspend {
			break
		}
	}

	if width == 0 && height == 0 {
		width = 70
		height = 7
	}

	textView := tview.NewTextView().
		SetText(desc).
		SetTextColor(gomu.accentColor)

	// debugLog(fmt.Sprintf("from color: %d", tcell.GetColor("#FF0000")))
	// debugLog(fmt.Sprintf("from color: %d", tcell.GetColor("#0A0F14")))
	// debugLog(fmt.Sprintf("from config: %d", gomu.popupBg))
	textView.SetTextAlign(tview.AlignCenter).SetBackgroundColor(gomu.popupBg)

	box := tview.NewFrame(textView).SetBorders(1, 0, 0, 0, 0, 0)
	box.SetTitle(title).SetBorder(true).SetBackgroundColor(gomu.popupBg)
	popupId := fmt.Sprintf("%s %d", "timeout-popup", popupCounter)

	popupCounter++
	gomu.pages.AddPage(popupId, topRight(box, width, height), true, true)
	gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))

	go func() {
		time.Sleep(timeout)
		gomu.pages.RemovePage(popupId)
		gomu.app.Draw()

		// timed popup shouldn't get focused
		// this here check if another popup exists and focus that instead of panel
		// if none continue focus panel
		topPopup := gomu.popups.peekTop()
		if topPopup == nil {
			gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))
		} else {
			gomu.app.SetFocus(topPopup)
		}
	}()
}

// Shows popup for the current volume
func volumePopup(volume float64) {

	currVol := volToHuman(volume)
	maxVol := 100
	// max progress bar length
	maxLength := 50

	progressBar := progresStr(currVol, maxVol, maxLength, "█", "-")

	progress := fmt.Sprintf("\n%d |%s| %d",
		currVol,
		progressBar,
		maxVol,
	)

	timedPopup(" Volume ", progress, getPopupTimeout(), 0, 0)
}

// Shows a list of keybind. The upper list is the local keybindings to specific
// panel only. The lower list is the global keybindings
func helpPopup(panel Panel) {

	helpText := panel.help()

	genHelp := []string{
		" ",
		"tab    change panel",
		"space  toggle play/pause",
		"esc    close popup",
		"n      skip",
		"q      quit",
		"+      volume up",
		"-      volume down",
		"?      toggle help",
	}

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(gomu.popupBg).SetTitle(" Help ").
		SetBorder(true)
	list.SetSelectedBackgroundColor(gomu.popupBg).
		SetSelectedTextColor(gomu.accentColor)

	for _, v := range append(helpText, genHelp...) {
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
			gomu.pages.RemovePage("help-page")
			gomu.popups.pop()
		}

		return nil
	})

	gomu.pages.AddPage("help-page", center(list, 50, 30), true, true)
	gomu.popups.push(list)
}

// Input popup. Takes video url from youtube to be downloaded
func downloadMusicPopup(selPlaylist *tview.TreeNode) {

	inputField := tview.NewInputField().
		SetLabel("Enter a url: ").
		SetFieldWidth(0).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50)).
		SetFieldBackgroundColor(gomu.accentColor).
		SetFieldTextColor(tcell.ColorBlack)

	inputField.SetBackgroundColor(gomu.popupBg).
		SetBorder(true).SetTitle(" Ytdl ")

	inputField.SetDoneFunc(func(key tcell.Key) {

		switch key {
		case tcell.KeyEnter:
			url := inputField.GetText()

			go func() {
				if err := ytdl(url, selPlaylist); err != nil {
					logError(err)
				}
			}()
			gomu.pages.RemovePage("download-input-popup")
			gomu.popups.pop()

		case tcell.KeyEscape:
			gomu.pages.RemovePage("download-input-popup")
			gomu.popups.pop()
		}

		gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))

	})

	gomu.pages.
		AddPage("download-input-popup", center(inputField, 50, 4), true, true)

	gomu.popups.push(inputField)

}

// Input popup that takes the name of directory to be created
func createPlaylistPopup() {

	inputField := tview.NewInputField().
		SetLabel("Enter a playlist name: ").
		SetFieldWidth(0).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50)).
		SetFieldBackgroundColor(gomu.accentColor).
		SetFieldTextColor(tcell.ColorBlack)

	inputField.
		SetBackgroundColor(gomu.popupBg).
		SetBorder(true).
		SetTitle(" New Playlist ")

	inputField.SetDoneFunc(func(key tcell.Key) {

		switch key {
		case tcell.KeyEnter:
			playListName := inputField.GetText()
			err := gomu.playlist.createPlaylist(playListName)

			if err != nil {
				logError(err)
			}

			gomu.pages.RemovePage("mkdir-input-popup")
			gomu.popups.pop()

		case tcell.KeyEsc:
			gomu.pages.RemovePage("mkdir-input-popup")
			gomu.popups.pop()
		}

	})

	gomu.pages.
		AddPage("mkdir-input-popup", center(inputField, 50, 4), true, true)

	gomu.popups.push(inputField)

}

func exitConfirmation(args Args) {

	confirmationPopup("Are you sure to exit?", func(_ int, label string) {

		if label == "no" || label == "" {
			return
		}

		err := gomu.quit(args)
		if err != nil {
			logError(err)
		}
	})
}

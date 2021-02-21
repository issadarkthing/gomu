// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
	"github.com/ztrue/tracerr"
)

// this is used to make the popup unique
// this mitigates the issue of closing all popups when timeout ends
var (
	popupCounter = 0
)

// Stack Simple stack data structure
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
	s.popups[len(s.popups)-1] = nil // avoid memory leak

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

	dur := gomu.anko.GetString("General.popup_timeout")
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
		SetBackgroundColor(gomu.colors.popup).
		AddButtons([]string{"no", "yes"}).
		SetButtonBackgroundColor(gomu.colors.popup).
		SetButtonTextColor(gomu.colors.accent).
		SetDoneFunc(func(indx int, label string) {
			gomu.pages.RemovePage("confirmation-popup")
			gomu.popups.pop()
			handler(indx, label)
		})

	modal.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Rune() {
		case 'h':
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		case 'j':
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
		case 'l':
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
		}
		return e
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

// Width and height parameter are optional, provide 0 for both to use deault values.
// It defaults to 70 and 7 respectively.
func timedPopup(
	title string, desc string, timeout time.Duration, width, height int,
) {

	if width == 0 && height == 0 {
		width = 70
		height = 7
	}

	textView := tview.NewTextView().
		SetText(desc).
		SetTextColor(gomu.colors.accent)

	textView.SetTextAlign(tview.AlignCenter).SetBackgroundColor(gomu.colors.popup)

	box := tview.NewFrame(textView).SetBorders(1, 0, 0, 0, 0, 0)
	box.SetTitle(title).SetBorder(true).SetBackgroundColor(gomu.colors.popup)
	popupID := fmt.Sprintf("%s %d", "timeout-popup", popupCounter)

	popupCounter++
	gomu.pages.AddPage(popupID, topRight(box, width, height), true, true)
	// gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))

	resetFocus := func() {
		// timed popup shouldn't get focused
		// this here check if another popup exists and focus that instead of panel
		// if none continue focus panel
		topPopup := gomu.popups.peekTop()
		if topPopup == nil {
			gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))
		} else {
			gomu.app.SetFocus(topPopup)
		}
	}

	resetFocus()

	go func() {
		time.Sleep(timeout)
		gomu.pages.RemovePage(popupID)
		gomu.app.Draw()

		resetFocus()
	}()
}

// Wrapper for timed popup
func defaultTimedPopup(title, description string) {
	timedPopup(title, description, getPopupTimeout(), 0, 0)
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

	defaultTimedPopup(" Volume ", progress)
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
		"f/F    forward 10/60 seconds",
		"b/B    rewind 10/60 seconds",
		"?      toggle help",
		"m      open repl",
	}

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(gomu.colors.popup).SetTitle(" Help ").
		SetBorder(true)
	list.SetSelectedBackgroundColor(gomu.colors.popup).
		SetSelectedTextColor(gomu.colors.accent)

	for _, v := range append(helpText, genHelp...) {
		list.AddItem("  "+v, "", 0, nil)
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

	re := regexp.MustCompile(`^((?:https?:)?\/\/)?((?:www|m)\.)?((?:youtube\.com|youtu.be))(\/(?:[\w\-]+\?v=|embed\/|v\/)?)([\w\-]+)(\S+)?$`)

	popupID := "download-input-popup"
	input := newInputPopup(popupID, " Download ", "Url: ", "")

	input.SetDoneFunc(func(key tcell.Key) {

		switch key {
		case tcell.KeyEnter:
			url := input.GetText()

			// check if valid youtube url was given
			if re.MatchString(url) {
				go func() {
					if err := ytdl(url, selPlaylist); err != nil {
						logError(err)
					}
				}()
			} else {
				defaultTimedPopup("Invalid url", "Invalid youtube url was given")
			}

			gomu.pages.RemovePage("download-input-popup")
			gomu.popups.pop()

		case tcell.KeyEscape:
			gomu.pages.RemovePage("download-input-popup")
			gomu.popups.pop()
		}

		gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))

	})

}

// Input popup that takes the name of directory to be created
func createPlaylistPopup() {

	popupID := "mkdir-input-popup"
	input := newInputPopup(popupID, " New Playlist ", "Enter playlist name: ", "")

	input.SetDoneFunc(func(key tcell.Key) {

		switch key {
		case tcell.KeyEnter:
			playListName := input.GetText()
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

func searchPopup(title string, stringsToMatch []string, handler func(selected string)) {

	list := tview.NewList().ShowSecondaryText(false)
	list.SetSelectedBackgroundColor(gomu.colors.accent)
	list.SetHighlightFullLine(true)

	for _, v := range stringsToMatch {
		list.AddItem(v, v, 0, nil)
	}

	input := tview.NewInputField()
	input.SetFieldBackgroundColor(gomu.colors.popup).
		SetLabel("[red]>[-] ")
	input.SetChangedFunc(func(text string) {

		list.Clear()

		// list all item if input is empty
		if len(text) == 0 {
			for _, v := range stringsToMatch {
				list.AddItem(v, v, 0, nil)
			}
			return
		}

		pattern := input.GetText()
		matches := fuzzy.Find(pattern, stringsToMatch)
		const highlight = "[red]%c[-]"
		// const highlight = "[red]%s[-]"

		for _, match := range matches {
			var text strings.Builder
			matchrune := []rune(match.Str)
			matchruneIndexes := match.MatchedIndexes
			for i := 0; i < len(match.MatchedIndexes); i++ {
				matchruneIndexes[i] =
					utf8.RuneCountInString(match.Str[0:match.MatchedIndexes[i]])
			}
			for i := 0; i < len(matchrune); i++ {
				if contains(i, matchruneIndexes) {
					textwithcolor := fmt.Sprintf(highlight, matchrune[i])
					for _, j := range textwithcolor {
						text.WriteRune(j)
					}
				} else {
					text.WriteRune(matchrune[i])
				}
			}
			list.AddItem(text.String(), match.Str, 0, nil)
		}
	})

	input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		switch e.Key() {
		case tcell.KeyCtrlN, tcell.KeyDown, tcell.KeyCtrlJ:
			currIndx := list.GetCurrentItem()
			// if last index
			if currIndx == list.GetItemCount()-1 {
				currIndx = 0
			} else {
				currIndx++
			}
			list.SetCurrentItem(currIndx)

		case tcell.KeyCtrlP, tcell.KeyUp, tcell.KeyCtrlK:
			currIndx := list.GetCurrentItem()

			if currIndx == 0 {
				currIndx = list.GetItemCount() - 1
			} else {
				currIndx--
			}
			list.SetCurrentItem(currIndx)

		case tcell.KeyEnter:
			if list.GetItemCount() > 0 {
				_, selected := list.GetItemText(list.GetCurrentItem())
				gomu.pages.RemovePage("search-input-popup")
				gomu.popups.pop()
				handler(selected)
			}

		case tcell.KeyEscape:
			gomu.pages.RemovePage("search-input-popup")
			gomu.popups.pop()
		}

		return e
	})

	popup := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(input, 2, 1, true).
		AddItem(list, 0, 1, false)

	popup.SetBorder(true).
		SetBackgroundColor(gomu.colors.popup).
		SetBorderPadding(1, 1, 2, 2).
		SetTitle(" " + title + " ")

	gomu.pages.AddPage("search-input-popup", center(popup, 70, 40), true, true)
	gomu.popups.push(popup)
}

// Creates new popup widget with default settings
func newInputPopup(popupID, title, label string, text string) *tview.InputField {

	inputField := tview.NewInputField().
		SetLabel(label).
		SetFieldWidth(0).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50)).
		SetFieldBackgroundColor(gomu.colors.popup).
		SetFieldTextColor(gomu.colors.foreground)

	inputField.SetBackgroundColor(gomu.colors.popup).
		SetTitle(title).
		SetBorder(true).
		SetBorderPadding(1, 0, 2, 2)

	inputField.SetText(text)

	gomu.pages.
		AddPage(popupID, center(inputField, 60, 5), true, true)

	gomu.popups.push(inputField)

	return inputField
}

func renamePopup(node *AudioFile) {

	popupID := "rename-input-popup"
	input := newInputPopup(popupID, " Rename ", "New name: ", node.name)
	input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		switch e.Key() {
		case tcell.KeyEnter:
			newName := input.GetText()
			if newName == "" {
				return e
			}
			err := gomu.playlist.rename(newName)
			if err != nil {
				defaultTimedPopup(" Error ", err.Error())
				logError(err)
			}
			gomu.pages.RemovePage(popupID)
			gomu.popups.pop()
			gomu.playlist.refresh()

			gomu.queue.updateQueueNames()
			gomu.setFocusPanel(gomu.playlist)
			gomu.prevPanel = gomu.playlist

			root := gomu.playlist.GetRoot()
			root.Walk(func(node, _ *tview.TreeNode) bool {
				if strings.Contains(node.GetText(), newName) {
					gomu.playlist.setHighlight(node)
				}
				return true
			})

		case tcell.KeyEsc:
			gomu.pages.RemovePage(popupID)
			gomu.popups.pop()
		}

		return e
	})
}

// Show error popup with error is logged. Prefer this when its related with user
// interaction. Otherwise, use logError.
func errorPopup(message error) {
	defaultTimedPopup(" Error ", tracerr.Unwrap(message).Error())
	logError(message)
}

// Show debug popup and log debug info. Mainly used when scripting.
func debugPopup(message interface{}) {
	m := fmt.Sprintf("%v", message)
	defaultTimedPopup(" Debug ", m)
	logDebug(m)
}

// Show info popup and does not log anything. Prefer this when making simple
// popup.
func infoPopup(message string) {
	defaultTimedPopup(" Info ", message)
}

func inputPopup(prompt string, handler func(string)) {

	popupID := "general-input-popup"
	input := newInputPopup(popupID, "", prompt+": ", "")
	input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		switch e.Key() {
		case tcell.KeyEnter:
			output := input.GetText()
			if output == "" {
				return e
			}
			handler(output)
			gomu.pages.RemovePage(popupID)
			gomu.popups.pop()

		case tcell.KeyEsc:
			gomu.pages.RemovePage(popupID)
			gomu.popups.pop()
		}

		return e
	})
}

func replPopup() {

	popupId := "repl-input-popup"
	prompt := "> "

	textview := tview.NewTextView()
	input := tview.NewInputField().
		SetFieldBackgroundColor(gomu.colors.popup).
		SetLabel(prompt)

	// to store input history
	history := []string{}
	upCount := 0

	gomu.anko.Define("println", func(x ...interface{}) {
		fmt.Fprintln(textview, x...)
	})
	gomu.anko.Define("print", func(x ...interface{}) {
		fmt.Fprint(textview, x...)
	})
	gomu.anko.Define("printf", func(format string, x ...interface{}) {
		fmt.Fprintf(textview, format, x...)
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		switch event.Key() {
		case tcell.KeyUp:

			if upCount < len(history) {
				input.SetText(history[upCount])
				upCount++
			}

		case tcell.KeyDown:

			if upCount > 0 {
				if upCount == len(history) {
					upCount -= 2
				} else {
					upCount -= 1
				}
				input.SetText(history[upCount])
			} else if upCount == 0 {
				input.SetText("")
			}

		case tcell.KeyCtrlL:
			textview.SetText("")

		case tcell.KeyEsc:
			gomu.pages.RemovePage(popupId)
			gomu.popups.pop()
			return nil

		case tcell.KeyEnter:
			text := input.GetText()
			// most recent is placed the most front
			history = append([]string{text}, history...)
			upCount = 0

			input.SetText("")

			defer func() {
				if err := recover(); err != nil {
					fmt.Fprintf(textview, "%s%s\n%v\n\n", prompt, text, err)
				}
			}()

			res, err := gomu.anko.Execute(text)
			if err != nil {
				fmt.Fprintf(textview, "%s%s\n%v\n\n", prompt, text, err)
				return nil
			} else {
				fmt.Fprintf(textview, "%s%s\n%v\n\n", prompt, text, res)
			}

		}

		return event
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(input, 3, 1, true).
		AddItem(textview, 0, 1, false)

	flex.
		SetBackgroundColor(gomu.colors.popup).
		SetBorder(true).
		SetBorderPadding(1, 1, 2, 2).
		SetTitle(" REPL ")

	gomu.pages.AddPage(popupId, center(flex, 90, 30), true, true)
	gomu.popups.push(flex)
}

func ytSearchPopup() {

	popupId := "youtube-search-input-popup"

	input := newInputPopup(popupId, " Youtube Search ", "search: ", "")


	var mutex sync.Mutex
	prefixMap := make(map[string][]string)

	// quick hack to change the autocomplete text color
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
	input.SetAutocompleteFunc(func(currentText string) []string {

		// Ignore empty text.
		prefix := strings.TrimSpace(strings.ToLower(currentText))
		if prefix == "" {
			return nil
		}

		mutex.Lock()
		defer mutex.Unlock()
		entries, ok := prefixMap[prefix]
		if ok {
			return entries
		}

		go func() {
			suggestions, err := getSuggestions(currentText)
			if err != nil {
				logError(err)
				return
			}

			mutex.Lock()
			prefixMap[prefix] = suggestions
			mutex.Unlock()

			input.Autocomplete()
			gomu.app.Draw()
		}()

		return nil
	})

	input.SetDoneFunc(func(key tcell.Key) {

		switch key {
		case tcell.KeyEnter:
			search := input.GetText()
			defaultTimedPopup(" Youtube Search ", "Searching for "+search)
			gomu.pages.RemovePage(popupId)
			gomu.popups.pop()

			go func() {

				results, err := getSearchResult(search)
				if err != nil {
					logError(err)
					defaultTimedPopup(" Error ", err.Error())
					return
				}

				titles := []string{}
				urls := make(map[string]string)

				for _, result := range results {
					duration, err := time.ParseDuration(fmt.Sprintf("%ds", result.LengthSeconds))
					if err != nil {
						logError(err)
						return
					}

					durationText := fmt.Sprintf("[ %s ] ", fmtDuration(duration))
					title := durationText + result.Title

					urls[title] = `https://www.youtube.com/watch?v=` + result.VideoId

					titles = append(titles, title)
				}

				searchPopup("Youtube Videos", titles, func(title string) {

					audioFile := gomu.playlist.getCurrentFile()

					var dir *tview.TreeNode

					if audioFile.isAudioFile {
						dir = audioFile.parent
					} else {
						dir = audioFile.node
					}

					go func() {
						url := urls[title]
						if err := ytdl(url, dir); err != nil {
							logError(err)
						}
						gomu.playlist.refresh()
					}()
					gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))
				})

				gomu.app.Draw()
			}()

		case tcell.KeyEscape:
			gomu.pages.RemovePage(popupId)
			gomu.popups.pop()
			gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))

		}

	})
}

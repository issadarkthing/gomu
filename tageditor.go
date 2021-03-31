// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/tramhao/id3v2"
	"github.com/ztrue/tracerr"

	"github.com/issadarkthing/gomu/lyric"
)

// lyricFlex extend the flex control to modify the Focus item
type lyricFlex struct {
	*tview.Flex
	FocusedItem tview.Primitive
	inputs      []tview.Primitive
	box         *tview.Box
}

// tagPopup is used to edit tag, delete and fetch lyrics
func tagPopup(node *AudioFile) (err error) {

	popupID := "tag-editor-input-popup"
	tag, popupLyricMap, options, err := node.loadTagMap()
	if err != nil {
		return tracerr.Wrap(err)
	}

	var (
		artistInputField  *tview.InputField = tview.NewInputField()
		titleInputField   *tview.InputField = tview.NewInputField()
		albumInputField   *tview.InputField = tview.NewInputField()
		getTagButton      *tview.Button     = tview.NewButton("Get Tag")
		saveTagButton     *tview.Button     = tview.NewButton("Save Tag")
		lyricDropDown     *tview.DropDown   = tview.NewDropDown()
		deleteLyricButton *tview.Button     = tview.NewButton("Delete Lyric")
		getLyricDropDown  *tview.DropDown   = tview.NewDropDown()
		getLyricButton    *tview.Button     = tview.NewButton("Fetch Lyric")
		lyricTextView     *tview.TextView   = tview.NewTextView()
		leftGrid          *tview.Grid       = tview.NewGrid()
		rightFlex         *tview.Flex       = tview.NewFlex()
	)

	artistInputField.SetLabel("Artist: ").
		SetFieldWidth(20).
		SetText(tag.Artist()).
		SetFieldBackgroundColor(gomu.colors.popup)

	titleInputField.SetLabel("Title:  ").
		SetFieldWidth(20).
		SetText(tag.Title()).
		SetFieldBackgroundColor(gomu.colors.popup)

	albumInputField.SetLabel("Album:  ").
		SetFieldWidth(20).
		SetText(tag.Album()).
		SetFieldBackgroundColor(gomu.colors.popup)

	leftBox := tview.NewBox().
		SetBorder(true).
		SetTitle(node.name).
		SetBackgroundColor(gomu.colors.popup).
		SetBorderColor(gomu.colors.accent).
		SetTitleColor(gomu.colors.accent).
		SetBorderPadding(1, 1, 2, 2)

	getTagButton.SetSelectedFunc(func() {
		var titles []string
		audioFile := node
		go func() {
			var getLyric lyric.GetLyricCn
			results, err := getLyric.GetLyricOptions(audioFile.name)
			if err != nil {
				errorPopup(err)
				return
			}
			for _, v := range results {
				titles = append(titles, v.TitleForPopup)
			}

			go func() {
				searchPopup(" Song Tags ", titles, func(selected string) {
					if selected == "" {
						return
					}

					var selectedIndex int
					for i, v := range results {
						if v.TitleForPopup == selected {
							selectedIndex = i
							break
						}
					}

					newTag := results[selectedIndex]
					artistInputField.SetText(newTag.Artist)
					titleInputField.SetText(newTag.Title)
					albumInputField.SetText(newTag.Album)

					tag, err = id3v2.Open(node.path, id3v2.Options{Parse: true})
					if err != nil {
						errorPopup(err)
						return
					}
					defer tag.Close()
					tag.SetArtist(newTag.Artist)
					tag.SetTitle(newTag.Title)
					tag.SetAlbum(newTag.Album)
					err = tag.Save()
					if err != nil {
						errorPopup(err)
						return
					}
					if gomu.anko.GetBool("General.rename_bytag") {
						newName := fmt.Sprintf("%s-%s", newTag.Artist, newTag.Title)
						err = gomu.playlist.rename(newName)
						if err != nil {
							errorPopup(err)
							return
						}
						gomu.playlist.refresh()
						leftBox.SetTitle(newName)
					}
					defaultTimedPopup(" Success ", "Tag update successfully")
				})
			}()
		}()
	}).
		SetBackgroundColorActivated(gomu.colors.popup).
		SetLabelColorActivated(gomu.colors.accent).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.popup).
		SetTitleColor(gomu.colors.accent)

	saveTagButton.SetSelectedFunc(func() {
		tag, err = id3v2.Open(node.path, id3v2.Options{Parse: true})
		if err != nil {
			errorPopup(err)
			return
		}
		defer tag.Close()
		newArtist := artistInputField.GetText()
		newTitle := titleInputField.GetText()
		newAlbum := albumInputField.GetText()
		tag.SetArtist(newArtist)
		tag.SetTitle(newTitle)
		tag.SetAlbum(newAlbum)
		err = tag.Save()
		if err != nil {
			errorPopup(err)
			return
		}
		if gomu.anko.GetBool("General.rename_bytag") {
			newName := fmt.Sprintf("%s-%s", newArtist, newTitle)
			err = gomu.playlist.rename(newName)
			if err != nil {
				errorPopup(err)
				return
			}
			gomu.playlist.refresh()
			leftBox.SetTitle(newName)
		}

		defaultTimedPopup(" Success ", "Tag update successfully")

	}).
		SetBackgroundColorActivated(gomu.colors.popup).
		SetLabelColorActivated(gomu.colors.accent).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.popup).
		SetTitleColor(gomu.colors.foreground)

	lyricDropDown.SetOptions(options, nil).
		SetCurrentOption(0).
		SetFieldBackgroundColor(gomu.colors.popup).
		SetFieldTextColor(gomu.colors.accent).
		SetPrefixTextColor(gomu.colors.accent).
		SetSelectedFunc(func(text string, _ int) {
			lyricTextView.SetText(popupLyricMap[text]).
				SetTitle(" " + text + " lyric preview ")
		}).
		SetLabel("Embeded Lyrics: ")
	lyricDropDown.SetBackgroundColor(gomu.colors.popup)

	deleteLyricButton.SetSelectedFunc(func() {
		_, langExt := lyricDropDown.GetCurrentOption()
		lyric := &lyric.Lyric{
			LangExt: langExt,
		}
		if len(options) > 0 {
			err := embedLyric(node.path, lyric, true)
			if err != nil {
				errorPopup(err)
				return
			}
			infoPopup(langExt + " lyric deleted successfully.")

			// Update map
			delete(popupLyricMap, langExt)

			// Update dropdown options
			var newOptions []string
			for _, v := range options {
				if v == langExt {
					continue
				}
				newOptions = append(newOptions, v)
			}
			options = newOptions
			lyricDropDown.SetOptions(newOptions, nil).
				SetCurrentOption(0).
				SetSelectedFunc(func(text string, _ int) {
					lyricTextView.SetText(popupLyricMap[text]).
						SetTitle(" " + text + " lyric preview ")
				})

				// Update lyric preview
			if len(newOptions) > 0 {
				_, langExt = lyricDropDown.GetCurrentOption()
				lyricTextView.SetText(popupLyricMap[langExt]).
					SetTitle(" " + langExt + " lyric preview ")
			} else {
				langExt = ""
				lyricTextView.SetText("No lyric embeded.").
					SetTitle(" " + langExt + " lyric preview ")
			}
		} else {
			infoPopup("No lyric embeded.")
		}
	}).
		SetBackgroundColorActivated(gomu.colors.popup).
		SetLabelColorActivated(gomu.colors.accent).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.popup).
		SetTitleColor(gomu.colors.accent)

	getLyricDropDownOptions := []string{"en", "zh-CN"}
	getLyricDropDown.SetOptions(getLyricDropDownOptions, nil).
		SetCurrentOption(0).
		SetFieldBackgroundColor(gomu.colors.popup).
		SetFieldTextColor(gomu.colors.accent).
		SetPrefixTextColor(gomu.colors.accent).
		SetLabel("Fetch Lyrics: ").
		SetBackgroundColor(gomu.colors.popup)

	langLyricFromConfig := gomu.anko.GetString("General.lang_lyric")
	if strings.Contains(langLyricFromConfig, "zh-CN") {
		getLyricDropDown.SetCurrentOption(1)
	}

	getLyricButton.SetSelectedFunc(func() {

		audioFile := gomu.playlist.getCurrentFile()
		_, lang := getLyricDropDown.GetCurrentOption()

		if !audioFile.isAudioFile {
			errorPopup(errors.New("not an audio file"))
			return
		}

		var wg sync.WaitGroup

		wg.Add(1)

		go func() {
			err := lyricPopup(lang, audioFile, &wg)
			if err != nil {
				errorPopup(err)
				return
			}
		}()

		go func() {
			// This is to ensure that the above go routine finish.
			wg.Wait()
			_, popupLyricMap, newOptions, err := audioFile.loadTagMap()
			if err != nil {
				errorPopup(err)
				gomu.app.Draw()
				return
			}

			options = newOptions
			// Update dropdown options
			gomu.app.QueueUpdateDraw(func() {
				lyricDropDown.SetOptions(newOptions, nil).
					SetCurrentOption(0).
					SetSelectedFunc(func(text string, _ int) {
						lyricTextView.SetText(popupLyricMap[text]).
							SetTitle(" " + text + " lyric preview ")
					})

				// Update lyric preview
				if len(newOptions) > 0 {
					_, langExt := lyricDropDown.GetCurrentOption()
					lyricTextView.SetText(popupLyricMap[langExt]).
						SetTitle(" " + langExt + " lyric preview ")
				} else {
					lyricTextView.SetText("No lyric embeded.").
						SetTitle(" lyric preview ")
				}
			})
		}()
	}).
		SetBackgroundColorActivated(gomu.colors.popup).
		SetLabelColorActivated(gomu.colors.accent).
		SetBorder(true).
		SetTitleColor(gomu.colors.accent).
		SetBackgroundColor(gomu.colors.popup)

	var lyricText string
	_, langExt := lyricDropDown.GetCurrentOption()
	lyricText = popupLyricMap[langExt]
	if lyricText == "" {
		lyricText = "No lyric embeded."
		langExt = ""
	}

	lyricTextView.
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true).
		SetTitle(" " + langExt + " lyric preview ").
		SetBorder(true)

	lyricTextView.SetText(lyricText).
		SetScrollable(true).
		SetWordWrap(true).
		SetWrap(true).
		SetBorder(true)
	lyricTextView.SetChangedFunc(func() {
		gomu.app.QueueUpdate(func() {
			lyricTextView.ScrollToBeginning()
		})
	})

	leftGrid.SetRows(3, 1, 3, 3, 3, 3, 0, 3, 3, 1, 3, 3).
		SetColumns(30).
		AddItem(getTagButton, 0, 0, 1, 3, 1, 10, true).
		AddItem(artistInputField, 2, 0, 1, 3, 1, 10, true).
		AddItem(titleInputField, 3, 0, 1, 3, 1, 10, true).
		AddItem(albumInputField, 4, 0, 1, 3, 1, 10, true).
		AddItem(saveTagButton, 5, 0, 1, 3, 1, 10, true).
		AddItem(getLyricDropDown, 7, 0, 1, 3, 1, 20, true).
		AddItem(getLyricButton, 8, 0, 1, 3, 1, 10, true).
		AddItem(lyricDropDown, 10, 0, 1, 3, 1, 10, true).
		AddItem(deleteLyricButton, 11, 0, 1, 3, 1, 10, true)

	rightFlex.SetDirection(tview.FlexColumn).
		AddItem(lyricTextView, 0, 1, true)

	lyricFlex := &lyricFlex{
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(leftGrid, 0, 2, true).
			AddItem(rightFlex, 0, 3, true),
		nil,
		nil,
		leftBox,
	}

	leftGrid.Box = lyricFlex.box

	lyricFlex.inputs = []tview.Primitive{
		getTagButton,
		artistInputField,
		titleInputField,
		albumInputField,
		saveTagButton,
		getLyricDropDown,
		getLyricButton,
		lyricDropDown,
		deleteLyricButton,
		lyricTextView,
	}

	gomu.pages.AddPage(popupID, center(lyricFlex, 90, 36), true, true)
	gomu.popups.push(lyricFlex)

	lyricFlex.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEnter:

		case tcell.KeyEsc:
			gomu.pages.RemovePage(popupID)
			gomu.popups.pop()
		case tcell.KeyTab, tcell.KeyCtrlN, tcell.KeyCtrlJ:
			lyricFlex.cycleFocus(gomu.app, false)
		case tcell.KeyBacktab, tcell.KeyCtrlP, tcell.KeyCtrlK:
			lyricFlex.cycleFocus(gomu.app, true)
		case tcell.KeyDown:
			lyricFlex.cycleFocus(gomu.app, false)
		case tcell.KeyUp:
			lyricFlex.cycleFocus(gomu.app, true)
		}

		switch e.Rune() {
		case 'q':
			if artistInputField.HasFocus() || titleInputField.HasFocus() || albumInputField.HasFocus() {
				return e
			}
			gomu.pages.RemovePage(popupID)
			gomu.popups.pop()
		}
		return e
	})

	return err
}

// This is a hack to cycle Focus in a flex
func (f *lyricFlex) cycleFocus(app *tview.Application, reverse bool) {
	for i, el := range f.inputs {
		if !el.HasFocus() {
			continue
		}

		if reverse {
			i = i - 1
			if i < 0 {
				i = len(f.inputs) - 1
			}
		} else {
			i = i + 1
			i = i % len(f.inputs)
		}

		app.SetFocus(f.inputs[i])
		f.FocusedItem = f.inputs[i]
		// below code is setting the border highlight of left and right flex
		if f.inputs[9].HasFocus() {
			f.inputs[9].(*tview.TextView).SetBorderColor(gomu.colors.accent).
				SetTitleColor(gomu.colors.accent)
			f.box.SetBorderColor(gomu.colors.background).
				SetTitleColor(gomu.colors.background)
		} else {
			f.inputs[9].(*tview.TextView).SetBorderColor(gomu.colors.background).
				SetTitleColor(gomu.colors.background)
			f.box.SetBorderColor(gomu.colors.accent).
				SetTitleColor(gomu.colors.accent)
		}
		return
	}
}

// Focus is an override of Focus function in tview.flex.
// This is to ensure that the focus of flex remain unchanged
// when returning from popups or search lists
func (f *lyricFlex) Focus(delegate func(p tview.Primitive)) {
	if f.FocusedItem != nil {
		gomu.app.SetFocus(f.FocusedItem)
	} else {
		f.Flex.Focus(delegate)
	}
}

// loadTagMap will load from tag and return a map of langExt to lyrics
func (a *AudioFile) loadTagMap() (tag *id3v2.Tag, popupLyricMap map[string]string, options []string, err error) {

	popupLyricMap = make(map[string]string)

	if a.isAudioFile {
		tag, err = id3v2.Open(a.path, id3v2.Options{Parse: true})
		if err != nil {
			return nil, nil, nil, tracerr.Wrap(err)
		}
		defer tag.Close()
	} else {
		return nil, nil, nil, fmt.Errorf("not an audio file")
	}
	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))

	for _, f := range usltFrames {
		uslf, ok := f.(id3v2.UnsynchronisedLyricsFrame)
		if !ok {
			die(errors.New("USLT error"))
		}
		res := uslf.Lyrics
		popupLyricMap[uslf.ContentDescriptor] = res
	}
	for option := range popupLyricMap {
		options = append(options, option)
	}
	sort.Strings(options)

	return tag, popupLyricMap, options, err
}

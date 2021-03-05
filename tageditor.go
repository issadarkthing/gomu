// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"fmt"
	"sort"

	"github.com/bogem/id3v2"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"

	"github.com/issadarkthing/gomu/lyric"
)

func tagPopup(node *AudioFile) (err error) {

	popupID := "tag-editor-input-popup"
	tag, popupLyricMap, options, err := loadTagMap(node)
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
		getLyric1Button   *tview.Button     = tview.NewButton("Get Lyric 1(en)")
		getLyric2Button   *tview.Button     = tview.NewButton("Get Lyric 2(zh-CN)")
		getLyric3Button   *tview.Button     = tview.NewButton("Get Lyric 3(zh-CN)")
		lyricTextView     *tview.TextView
		leftGrid          *tview.Grid = tview.NewGrid()
		rightFlex         *tview.Flex = tview.NewFlex()
	)
	artistInputField.SetLabel("Artist: ").
		SetFieldWidth(20).
		SetText(tag.Artist()).
		SetFieldBackgroundColor(gomu.colors.popup).
		SetBackgroundColor(gomu.colors.background)
	titleInputField.SetLabel("Title:  ").
		SetFieldWidth(20).
		SetText(tag.Title()).
		SetFieldBackgroundColor(gomu.colors.popup).
		SetBackgroundColor(gomu.colors.background)
	albumInputField.SetLabel("Album:  ").
		SetFieldWidth(20).
		SetText(tag.Album()).
		SetFieldBackgroundColor(gomu.colors.popup).
		SetBackgroundColor(gomu.colors.background)
	getTagButton.SetSelectedFunc(func() {
		audioFile := node
		serviceProvider := "netease"
		results, resultsTag, err := lyric.GetLyricOptionsChinese(audioFile.name, serviceProvider)
		if err != nil {
			errorPopup(err)
			return
		}

		titles := make([]string, 0, len(results))

		for result := range results {
			titles = append(titles, result)
		}

		searchPopup(" Lyrics ", titles, func(selected string) {
			if selected == "" {
				return
			}

			lyricID := results[selected]
			newTag := resultsTag[lyricID]
			artistInputField.SetText(newTag.Artist)
			titleInputField.SetText(newTag.Title)
			albumInputField.SetText(newTag.Album)
			tag, err = id3v2.Open(node.path, id3v2.Options{Parse: true})
			if err != nil {
				errorPopup(err)
			}
			defer tag.Close()
			tag.SetArtist(artistInputField.GetText())
			tag.SetTitle(titleInputField.GetText())
			tag.SetAlbum(albumInputField.GetText())
			err = tag.Save()
			if err != nil {
				errorPopup(err)
			} else {
				defaultTimedPopup(" Success ", "Tag update successfully")
			}
		})
	}).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.accent)
	saveTagButton.SetSelectedFunc(func() {
		tag, err = id3v2.Open(node.path, id3v2.Options{Parse: true})
		if err != nil {
			errorPopup(err)
		}
		defer tag.Close()
		tag.SetArtist(artistInputField.GetText())
		tag.SetTitle(titleInputField.GetText())
		tag.SetAlbum(albumInputField.GetText())
		err := tag.Save()
		if err != nil {
			errorPopup(err)
		} else {
			defaultTimedPopup(" Success ", "Tag update successfully")
		}
	}).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.foreground)

	deleteLyricButton.SetSelectedFunc(func() {
		_, langExt := lyricDropDown.GetCurrentOption()
		if len(options) > 0 {
			err := embedLyric(node.path, "", langExt, true)
			if err != nil {
				errorPopup(err)
			} else {
				infoPopup(langExt + " lyric deleted successfully.")
			}
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
				SetCurrentOption(0)
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
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.accent)

	getLyric1Button.SetSelectedFunc(func() {
		audioFile := node

		if audioFile.isAudioFile {
			go func() {
				gomu.app.QueueUpdateDraw(func() {
					results, err := lyric.GetLyricOptions(audioFile.name)
					if err != nil {
						errorPopup(err)
						gomu.app.Draw()
						return

					}

					titles := make([]string, 0, len(results))

					for result := range results {
						titles = append(titles, result)
					}

					searchPopup(" Lyrics ", titles, func(selected string) {
						if selected == "" {
							return
						}

						go func() {
							url := results[selected]
							lyric, err := lyric.GetLyric(url)
							if err != nil {
								errorPopup(err)
								gomu.app.Draw()
							}

							langExt := "en"
							err = embedLyric(audioFile.path, lyric, langExt, false)
							if err != nil {
								errorPopup(err)
								gomu.app.Draw()
							} else {
								infoPopup("en Lyric added successfully")
								gomu.app.Draw()
							}

							// This is to ensure that the above go routine finish.
							gomu.app.QueueUpdateDraw(func() {
								_, popupLyricMap, newOptions, err := loadTagMap(node)
								if err != nil {
									errorPopup(err)
									gomu.app.Draw()
									return
								}

								// Update dropdown options
								lyricDropDown.SetOptions(newOptions, nil).
									SetCurrentOption(0)
								// Update lyric preview
								if len(newOptions) > 0 {
									_, langExt := lyricDropDown.GetCurrentOption()
									lyricTextView.SetText(popupLyricMap[langExt]).
										SetTitle(" " + langExt + " lyric preview ")
									infoPopup(langExt + " lyric embeded successfully.")
								} else {
									lyricTextView.SetText("No lyric embeded.").
										SetTitle(" lyric preview ")
								}
							})
						}()
					})
				})
			}()
		}
	}).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.accent)
	getLyric2Button.SetSelectedFunc(func() {
		audioFile := node
		serviceProvider := "netease"
		results, _, err := lyric.GetLyricOptionsChinese(audioFile.name, serviceProvider)
		if err != nil {
			errorPopup(err)
			return
		}

		titles := make([]string, 0, len(results))

		for result := range results {
			titles = append(titles, result)
		}

		searchPopup(" Lyrics ", titles, func(selected string) {
			if selected == "" {
				return
			}

			go func() {
				lyricID := results[selected]
				lyric, err := lyric.GetLyricChinese(lyricID, serviceProvider)
				if err != nil {
					errorPopup(err)
					gomu.app.Draw()
					return
				}

				langExt := "zh-CN"
				err = embedLyric(audioFile.path, lyric, langExt, false)
				if err != nil {
					errorPopup(err)
					gomu.app.Draw()
				} else {
					infoPopup("cn Lyric added successfully")
					gomu.app.Draw()
				}
				// This is to ensure that the above go routine finish.
				gomu.app.QueueUpdateDraw(func() {
					_, popupLyricMap, newOptions, err := loadTagMap(node)
					if err != nil {
						errorPopup(err)
						gomu.app.Draw()
						return
					}
					options = newOptions

					// Update dropdown options
					lyricDropDown.SetOptions(newOptions, nil).
						SetCurrentOption(0)
					// Update lyric preview
					if len(newOptions) > 0 {
						_, langExt := lyricDropDown.GetCurrentOption()
						lyricTextView.SetText(popupLyricMap[langExt]).
							SetTitle(" " + langExt + " lyric preview ")
						infoPopup(langExt + " lyric embeded successfully.")
					} else {
						lyricTextView.SetText("No lyric embeded.").
							SetTitle(" lyric preview ")
					}
				})
			}()
		})
	}).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.accent)

	getLyric3Button.SetSelectedFunc(func() {
		audioFile := node
		serviceProvider := "kugou"
		results, _, err := lyric.GetLyricOptionsChinese(audioFile.name, serviceProvider)
		if err != nil {
			errorPopup(err)
			return
		}

		titles := make([]string, 0, len(results))

		for result := range results {
			titles = append(titles, result)
		}

		searchPopup(" Lyrics ", titles, func(selected string) {
			if selected == "" {
				return
			}

			go func() {
				lyricID := results[selected]
				lyric, err := lyric.GetLyricChinese(lyricID, serviceProvider)
				if err != nil {
					errorPopup(err)
					gomu.app.Draw()
					return
				}

				langExt := "zh-CN"
				err = embedLyric(audioFile.path, lyric, langExt, false)
				if err != nil {
					errorPopup(err)
					gomu.app.Draw()
				} else {
					infoPopup("cn Lyric added successfully")
					gomu.app.Draw()
				}
				// This is to ensure that the above go routine finish.
				gomu.app.QueueUpdateDraw(func() {
					_, popupLyricMap, newOptions, err := loadTagMap(node)
					if err != nil {
						errorPopup(err)
						gomu.app.Draw()
						return
					}

					// Update dropdown options
					lyricDropDown.SetOptions(newOptions, nil).
						SetCurrentOption(0)
					// Update lyric preview
					if len(newOptions) > 0 {
						_, langExt := lyricDropDown.GetCurrentOption()
						lyricTextView.SetText(popupLyricMap[langExt]).
							SetTitle(" " + langExt + " lyric preview ")
						infoPopup(langExt + " lyric embeded successfully.")
					} else {
						lyricTextView.SetText("No lyric embeded.").
							SetTitle(" lyric preview ")
					}
				})
			}()
		})
	}).
		SetBorder(true).
		SetBackgroundColor(gomu.colors.background).
		SetTitleColor(gomu.colors.accent)
	lyricDropDown.SetOptions(options, nil).
		SetCurrentOption(0).
		SetFieldBackgroundColor(gomu.colors.background).
		SetFieldTextColor(gomu.colors.accent).
		SetPrefixTextColor(gomu.colors.accent).
		SetSelectedFunc(func(text string, _ int) {
			lyricTextView.SetText(popupLyricMap[text]).
				SetTitle(" " + text + " lyric preview ")
		}).
		SetLabel("Embeded Lyrics: ")
	lyricDropDown.SetBackgroundColor(gomu.colors.popup)

	var lyricText string
	_, langExt := lyricDropDown.GetCurrentOption()
	lyricText = popupLyricMap[langExt]
	if lyricText == "" {
		lyricText = "No lyric embeded."
		langExt = ""
	}
	lyricTextView = tview.NewTextView()
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
		SetBackgroundColor(gomu.colors.popup).
		SetBorder(true)

	leftGrid.SetRows(3, 3, 3, 3, 3, 0, 3, 3, 3, 3, 3).
		SetColumns(30).
		AddItem(artistInputField, 0, 0, 1, 3, 1, 10, true).
		AddItem(titleInputField, 1, 0, 1, 3, 1, 10, true).
		AddItem(albumInputField, 2, 0, 1, 3, 1, 10, true).
		AddItem(getTagButton, 3, 0, 1, 3, 1, 10, true).
		AddItem(saveTagButton, 4, 0, 1, 3, 1, 10, true).
		AddItem(lyricDropDown, 6, 0, 1, 3, 1, 10, true).
		AddItem(deleteLyricButton, 7, 0, 1, 3, 1, 10, true).
		AddItem(getLyric1Button, 8, 0, 1, 3, 1, 10, true).
		AddItem(getLyric2Button, 9, 0, 1, 3, 1, 10, true).
		AddItem(getLyric3Button, 10, 0, 1, 3, 1, 10, true)
	leftGrid.SetBorder(true).
		SetTitle(node.name).
		SetBorderPadding(1, 1, 2, 2)

	rightFlex.SetDirection(tview.FlexColumn).
		AddItem(lyricTextView, 0, 1, true)

	lyricFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftGrid, 0, 2, true).
		AddItem(rightFlex, 0, 3, true)

	lyricFlex.
		SetTitle(node.name).
		SetBorderPadding(1, 1, 2, 2).
		SetBackgroundColor(gomu.colors.popup)

	inputs := []tview.Primitive{
		artistInputField,
		titleInputField,
		albumInputField,
		getTagButton,
		saveTagButton,
		lyricDropDown,
		deleteLyricButton,
		getLyric1Button,
		getLyric2Button,
		getLyric3Button,
		lyricTextView,
	}

	gomu.pages.
		AddPage(popupID, center(lyricFlex, 90, 40), true, true)
	gomu.popups.push(lyricFlex)

	lyricFlex.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEnter:

		case tcell.KeyEsc:
			gomu.pages.RemovePage(popupID)
			gomu.popups.pop()
		case tcell.KeyTab:
			cycleFocus(gomu.app, inputs, false)
		case tcell.KeyBacktab:
			cycleFocus(gomu.app, inputs, true)
		case tcell.KeyDown:
			cycleFocus(gomu.app, inputs, false)
		case tcell.KeyUp:
			cycleFocus(gomu.app, inputs, true)
		}

		switch e.Rune() {
		case '1':
		case '2':
		case '3':
		}
		return e
	})

	return err
}

func cycleFocus(app *tview.Application, elements []tview.Primitive, reverse bool) {
	for i, el := range elements {
		if !el.HasFocus() {
			continue
		}

		if reverse {
			i = i - 1
			if i < 0 {
				i = len(elements) - 1
			}
		} else {
			i = i + 1
			i = i % len(elements)
		}

		app.SetFocus(elements[i])
		return
	}
}

func loadTagMap(node *AudioFile) (tag *id3v2.Tag, popupLyricMap map[string]string, options []string, err error) {

	popupLyricMap = make(map[string]string)

	if node.isAudioFile {
		tag, err = id3v2.Open(node.path, id3v2.Options{Parse: true})
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

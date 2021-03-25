package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Colors are the configurable colors used in gomu
type Colors struct {
	accent     tcell.Color
	background tcell.Color
	foreground tcell.Color
	// title refers to now_playing_title in config file
	title       tcell.Color
	popup       tcell.Color
	playlistHi  tcell.Color
	playlistDir tcell.Color
	queueHi     tcell.Color
	subtitle    string
}

func init() {
	tcell.ColorNames["none"] = tcell.ColorDefault
}

func newColor() *Colors {

	defaultColors := map[string]string{
		"Color.accent":             "darkcyan",
		"Color.background":         "none",
		"Color.foreground":         "white",
		"Color.popup":              "black",
		"Color.playlist_directory": "darkcyan",
		"Color.playlist_highlight": "darkcyan",
		"Color.queue_highlight":    "darkcyan",
		"Color.now_playing_title":  "darkgreen",
		"Color.subtitle":           "darkgoldenrod",
	}

	anko := gomu.anko

	// checks for invalid color and set default fallback
	for k, v := range defaultColors {

		// color from the config file
		cfgColor := anko.GetString(k)

		if _, ok := tcell.ColorNames[cfgColor]; !ok {
			// use default value if invalid hex color was given
			anko.Set(k, v)
		}
	}

	accent := anko.GetString("Color.accent")
	background := anko.GetString("Color.background")
	foreground := anko.GetString("Color.foreground")
	popup := anko.GetString("Color.popup")
	playlistDir := anko.GetString("Color.playlist_directory")
	playlistHi := anko.GetString("Color.playlist_highlight")
	queueHi := anko.GetString("Color.queue_highlight")
	title := anko.GetString("Color.now_playing_title")
	subtitle := anko.GetString("Color.subtitle")

	color := &Colors{
		accent:      tcell.ColorNames[accent],
		foreground:  tcell.ColorNames[foreground],
		background:  tcell.ColorNames[background],
		popup:       tcell.ColorNames[popup],
		playlistDir: tcell.ColorNames[playlistDir],
		playlistHi:  tcell.ColorNames[playlistHi],
		queueHi:     tcell.ColorNames[queueHi],
		title:       tcell.ColorNames[title],
		subtitle:    subtitle,
	}
	return color
}

func colorsPopup() tview.Primitive {

	textView := tview.NewTextView().
		SetWrap(true).
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true)

	textView.
		SetBorder(true).
		SetTitle(" Colors ").
		SetBorderPadding(1, 1, 2, 2)

	i := 0
	colorPad := strings.Repeat(" ", 5)

	for name := range tcell.ColorNames {
		fmt.Fprintf(textView, "%20s [:%s]%s[:-] ", name, name, colorPad)

		if i == 2 {
			fmt.Fprint(textView, "\n")
			i = 0
			continue
		}
		i++
	}

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			gomu.pages.RemovePage("show-color-popup")
			gomu.popups.pop()
		}
		return event
	})

	return textView
}

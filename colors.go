package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Colors struct {
	accent     tcell.Color
	foreground tcell.Color
	background tcell.Color
	// title refers to now_playing_title in config file
	title    tcell.Color
	popup    tcell.Color
	playlist tcell.Color
}

func init() {
	tcell.ColorNames["none"] = tcell.ColorDefault
}

func newColor() *Colors {

	defaultColors := map[string]string{
		"Color.accent":            "darkcyan",
		"Color.background":        "none",
		"Color.foreground":        "white",
		"Color.now_playing_title": "darkgreen",
		"Color.playlist":          "white",
		"Color.popup":             "black",
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
	foreground := anko.GetString("Color.foreground")
	background := anko.GetString("Color.background")
	popup := anko.GetString("Color.popup")
	title := anko.GetString("Color.now_playing_title")
	playlist := anko.GetString("Color.playlist")

	color := &Colors{
		accent:     tcell.ColorNames[accent],
		foreground: tcell.ColorNames[foreground],
		background: tcell.ColorNames[background],
		popup:      tcell.ColorNames[popup],
		title:      tcell.ColorNames[title],
		playlist:   tcell.ColorNames[playlist],
	}
	return color
}

func isValidColor(x tcell.Color) bool {
	return (x == tcell.ColorDefault) || x >= tcell.ColorBlack && x <= tcell.ColorYellowGreen
}

func intToColor(x int) tcell.Color {
	
	if x == -1 {
		return tcell.ColorDefault
	}

	return tcell.Color(x) + tcell.ColorBlack
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

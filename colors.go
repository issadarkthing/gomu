package main

import (
	"github.com/gdamore/tcell/v2"
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

func newColor() *Colors {

	defaultColors := map[string]string{
		"color_accent":            "#008B8B",
		"color_foreground":        "#FFFFFF",
		"color_background":        "none",
		"color_popup":             "#0A0F14",
		"color_now_playing_title": "#017702",
		"color_playlist":          "#008B8B",
	}

	anko := gomu.anko

	// Validate hex color
	for k, v := range defaultColors {

		// color from the config file
		cfgColor := anko.getString(k)
		if validHexColor(cfgColor) {
			continue
		}

		// use default value if invalid hex color was given
		anko.set(k, v)
	}

	// handle none background color
	var bgColor tcell.Color
	bg := anko.getString("color_background")

	if bg == "none" {
		bgColor = tcell.ColorDefault
	} else {
		bgColor = tcell.GetColor(bg)
	}

	accent := anko.getString("color_accent")
	foreground := anko.getString("color_foreground")
	popup := anko.getString("color_popup")
	title := anko.getString("color_now_playing_title")
	playlist := anko.getString("color_playlist")

	color := &Colors{
		accent:     tcell.GetColor(accent),
		foreground: tcell.GetColor(foreground),
		background: bgColor,
		popup:      tcell.GetColor(popup),
		title:      tcell.GetColor(title),
		playlist:   tcell.GetColor(playlist),
	}
	return color
}

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

	// Validate hex color
	for k, v := range defaultColors {

		// color from the config file
		cfgColor := getString(gomu.env, k)
		if validHexColor(cfgColor) {
			continue
		}

		// use default value if invalid hex color was given
		gomu.env.Set(k, v)
	}

	// handle none background color
	var bgColor tcell.Color
	bg := getString(gomu.env, "color_background")

	if bg == "none" {
		bgColor = tcell.ColorDefault
	} else {
		bgColor = tcell.GetColor(bg)
	}

	accent := getString(gomu.env, "color_accent")
	foreground := getString(gomu.env, "color_foreground")
	popup := getString(gomu.env, "color_popup")
	title := getString(gomu.env, "color_now_playing_title")
	playlist := getString(gomu.env, "color_playlist")

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

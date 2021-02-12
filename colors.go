package main

import (
	"log"

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
		cfgColor, err := getString(gomu.env, k)
		if err != nil {
			log.Fatal(err)
		}

		if validHexColor(cfgColor) {
			continue
		}

		// use default value if invalid hex color was given
		gomu.env.Set(k, v)
	}

	// handle none background color
	var bgColor tcell.Color
	bg, err := getString(gomu.env, "color_background")
	if err != nil {
		log.Fatal(err)
	}

	if bg == "none" {
		bgColor = tcell.ColorDefault
	} else {
		bgColor = tcell.GetColor(bg)
	}

	accent, err := getString(gomu.env, "color_accent")
	if err != nil {
		log.Fatal(err)
	}

	foreground, err := getString(gomu.env, "color_foreground")
	if err != nil {
		log.Fatal(err)
	}

	popup, err := getString(gomu.env, "color_popup")
	if err != nil {
		log.Fatal(err)
	}

	title, err := getString(gomu.env, "color_now_playing_title")
	if err != nil {
		log.Fatal(err)
	}

	playlist, err := getString(gomu.env, "color_playlist")
	if err != nil {
		log.Fatal(err)
	}

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

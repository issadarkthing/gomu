package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/spf13/viper"
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
		"color.accent":            "#008B8B",
		"color.foreground":        "#FFFFFF",
		"color.background":        "none",
		"color.popup":             "#0A0F14",
		"color.now_playing_title": "#017702",
		"color.playlist":          "#008B8B",
	}

	// Validate hex color
	for k, v := range defaultColors {

		// color from the config file
		cfgColor := viper.GetString(k)
		if validHexColor(cfgColor) {
			continue
		}

		// use default value if invalid hex color was given
		viper.Set(k, v)
	}

	// handle none background color
	var bgColor tcell.Color
	bg := viper.GetString("color.background")
	if bg == "none" {
		bgColor = tcell.ColorDefault
	} else {
		bgColor = tcell.GetColor(bg)
	}

	color := &Colors{
		accent:     tcell.GetColor(viper.GetString("color.accent")),
		foreground: tcell.GetColor(viper.GetString("color.foreground")),
		background: bgColor,
		popup:      tcell.GetColor(viper.GetString("color.popup")),
		title:      tcell.GetColor(viper.GetString("color.now_playing_title")),
		playlist:   tcell.GetColor(viper.GetString("color.playlist")),
	}
	return color
}

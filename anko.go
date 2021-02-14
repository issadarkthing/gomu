package main

import (
	"io/ioutil"
	"os"

	"github.com/mattn/anko/core"
	"github.com/mattn/anko/env"
	"github.com/mattn/anko/vm"
	_ "github.com/mattn/anko/packages"
	"github.com/ztrue/tracerr"
)


type Anko struct {
	env *env.Env
}

func newAnko() Anko {
	return Anko{
		core.Import(env.NewEnv()),
	}
}

// define defines new symbol and value to the Anko env
func (a *Anko) define(symbol string, value interface{}) error {
	return a.env.DefineGlobal(symbol, value)
}

// set sets new value to existing symbol. Use this when change value under an
// existing symbol.
func (a *Anko) set(symbol string, value interface{}) error {
	return a.env.Set(symbol, value)
}

// get gets value from anko env, returns error if symbol is not found.
func (a *Anko) get(symbol string) (interface{}, error) {
	return a.env.Get(symbol)
}

// getInt gets int value from symbol, returns golang default value if not found
func (a *Anko) getInt(symbol string) int {
	v, err := a.env.Get(symbol)
	if err != nil {
		return 0
	}

	val, ok := v.(int64)
	if !ok {
		return 0
	}

	return int(val)
}

// getString gets string value from symbol, returns golang default value if not
// found
func (a *Anko) getString(symbol string) string {
	v, err := a.env.Get(symbol)
	if err != nil {
		return ""
	}

	val, ok := v.(string)
	if !ok {
		return ""
	}

	return val
}

// getBool gets bool value from symbol, returns golang default value if not
// found
func (a *Anko) getBool(symbol string) bool {
	v, err := a.env.Get(symbol)
	if err != nil {
		return false
	}

	val, ok := v.(bool)
	if !ok {
		return false
	}

	return val
}

// execute executes anko script
func (a *Anko) execute(src string) (interface{}, error) {
	return vm.Execute(a.env, nil, src)
}

// executes user config with default config is executed first in order to apply
// default values
func execConfig(config string) error {

	const defaultConfig = `

// confirmation popup to add the whole playlist to the queue
confirm_bulk_add    = true
confirm_on_exit     = true
queue_loop          = false
load_prev_queue     = true
popup_timeout       = "5s"
// change this to directory that contains mp3 files
music_dir           = "~/music"
// url history of downloaded audio will be saved here
history_path        = "~/.local/share/gomu/urls"
// some of the terminal supports unicode character
// you can set this to true to enable emojis
use_emoji           = true
// initial volume when gomu starts up
volume              = 80
// if you experiencing error using this invidious instance, you can change it
// to another instance from this list:
// https://github.com/iv-org/documentation/blob/master/Invidious-Instances.md
invidious_instance  = "https://vid.puffyan.us"

// default emoji here is using awesome-terminal-fonts
// you can change these to your liking
emoji_playlist     = ""
emoji_file         = ""
emoji_loop         = "ﯩ"
emoji_noloop       = ""

// not all colors can be reproducible in terminal
// changing hex colors may or may not produce expected result
color_accent            = "#008B8B"
color_background        = "none"
color_foreground        = "#FFFFFF"
color_now_playing_title = "#017702"
color_playlist          = "#008B8B"
color_popup             = "#0A0F14"

// vim: syntax=go
`

	// built-in functions
	gomu.anko.define("debug_popup", debugPopup)
	gomu.anko.define("input_popup", inputPopup)
	gomu.anko.define("show_popup", defaultTimedPopup)
	gomu.anko.define("shell", shell)

	cfg := expandTilde(config)

	_, err := os.Stat(cfg)
	if os.IsNotExist(err) {
		err = appendFile(cfg, defaultConfig)
		if err != nil {
			return tracerr.Wrap(err)
		}
	}

	content, err := ioutil.ReadFile(cfg)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// execute default config
	_, err = gomu.anko.execute(defaultConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// execute user config
	_, err = gomu.anko.execute(string(content))
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

// Panel is used to keep track of childrens in slices
type Panel interface {
	HasFocus() bool
	SetBorderColor(color tcell.Color) *tview.Box
	SetTitleColor(color tcell.Color) *tview.Box
	SetTitle(s string) *tview.Box
	GetTitle() string
	help() []string
}

const (
	configPath  = ".config/gomu/config"
	historyPath = "~/.local/share/gomu/urls"
	musicPath   = "~/music"
)

// Reads config file and sets the options
func readConfig(args Args) {

	const config = `
general:
  # confirmation popup to add the whole playlist to the queue
  confirm_bulk_add:   true
  confirm_on_exit:    true
  load_prev_queue:    true
  queue_loop:         true
  # change this to directory that contains mp3 files
  music_dir:          ~/music
  # url history of downloaded audio will be saved here
  history_path:       ~/.local/share/gomu/urls
  popup_timeout:      5s
  # initial volume when gomu starts up
  volume:             100
  # some of the terminal supports unicode character
  # you can set this to true to enable emojis
  emoji:              false
  # you may use fzf as your finder inside gomu
  # but it is recommended to use built-in finder
  # as it integrates well with gomu
  fzf:                false
  # if you experiencing error using this invidious instance, you can change it
  # to another instance from this list:
  # https://github.com/iv-org/documentation/blob/master/Invidious-Instances.md
  invidious_instance: "https://vid.puffyan.us"

# not all colors can be reproducible in terminal
# changing hex colors may or may not produce expected result
color:
  accent:            "#008B8B"
  # none is transparent
  # only background has none attribute
  background:        none
  foreground:        "#FFFFFF"
  now_playing_title: "#017702"
  playlist:          "#008B8B"
  popup:             "#0A0F14"

# default emoji here is using awesome-terminal-fonts
# you can change these to your liking
emoji:
  playlist:          
  file:              
  loop:              ﯩ
  noloop:            
 
# vi:ft=yaml
`

	// config path passed by flag
	configPath := *args.config
	home, err := os.UserHomeDir()

	if err != nil {
		logError(err)
	}

	defaultPath := path.Join(home, configPath)

	if err != nil {
		logError(err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(strings.TrimSuffix(expandFilePath(configPath), "/config"))
	viper.AddConfigPath("$HOME/.config/gomu")

	// General config
	viper.SetDefault("general.music_dir", musicPath)
	viper.SetDefault("general.history_path", historyPath)
	viper.SetDefault("general.confirm_on_exit", true)
	viper.SetDefault("general.confirm_bulk_add", true)
	viper.SetDefault("general.popup_timeout", "5s")
	viper.SetDefault("general.volume", 100)
	viper.SetDefault("general.load_prev_queue", true)
	viper.SetDefault("general.use_emoji", false)
	viper.SetDefault("general.invidious_instance", "https://vid.puffyan.us")

	if err := viper.ReadInConfig(); err != nil {

		// creates gomu config dir if does not exist
		if _, err := os.Stat(defaultPath); err != nil {
			if err := os.MkdirAll(home+"/.config/gomu", 0755); err != nil {
				logError(err)
			}
		}

		// if config file was not found
		// copy default config to default config path
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {

			err = ioutil.WriteFile(defaultPath, []byte(config), 0644)
			if err != nil {
				logError(err)
			}

		}

	}
}

type Args struct {
	config  *string
	empty   *bool
	music   *string
	version *bool
}

func getArgs() Args {
	ar := Args{
		config:  flag.String("config", configPath, "Specify config file"),
		empty:   flag.Bool("empty", false, "Open gomu with empty queue. Does not override previous queue"),
		music:   flag.String("music", musicPath, "Specify music directory"),
		version: flag.Bool("version", false, "Print gomu version"),
	}
	flag.Parse()
	return ar
}

// Sets the layout of the application
func layout(gomu *Gomu) *tview.Flex {
	flex := tview.NewFlex().
		AddItem(gomu.playlist, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(gomu.queue, 0, 5, false).
			AddItem(gomu.playingBar, 0, 1, false), 0, 3, false)

	return flex
}

// Initialize
func start(application *tview.Application, args Args) {

	// Print version and exit
	if *args.version {
		fmt.Printf("Gomu %s\n", VERSION)
		return
	}

	// Assigning to global variable gomu
	gomu = newGomu()
	gomu.args = args

	// override default border
	// change double line border to one line border when focused
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
	tview.Styles.PrimitiveBackgroundColor = gomu.colors.background

	gomu.initPanels(application, args)
	gomu.command.defineCommands()

	flex := layout(gomu)
	gomu.pages.AddPage("main", flex, true, true)

	// sets the first focused panel
	gomu.setFocusPanel(gomu.playlist)
	gomu.prevPanel = gomu.playlist

	gomu.player.isLoop = viper.GetBool("general.queue_loop")
	gomu.queue.isLoop = gomu.player.isLoop

	if !*args.empty && viper.GetBool("general.load_prev_queue") {
		// load saved queue from previous session
		if err := gomu.queue.loadQueue(); err != nil {
			logError(err)
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		errMsg := fmt.Sprintf("Received %s. Exiting program", sig.String())
		logError(errors.New(errMsg))
		err := gomu.quit(args)
		if err != nil {
			logError(errors.New("unable to quit program"))
		}
	}()

	// global keybindings are handled here
	application.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		popupName, _ := gomu.pages.GetFrontPage()

		// disables keybindings when writing in input fields
		if strings.Contains(popupName, "-input-") {
			return e
		}

		switch e.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			if strings.Contains(popupName, "confirmation-") {
				return e
			}
			gomu.cyclePanels2()
		}

		cmds := map[rune]string{
			'q': "quit",
			' ': "toggle_pause",
			'+': "volume_up",
			'=': "volume_up",
			'-': "volume_down",
			'_': "volume_down",
			'n': "skip",
			':': "command_search",
			'?': "toggle_help",
			'f': "forward",
			'F': "forward_fast",
			'b': "rewind",
			'B': "rewind_fast",
		}

		for key, cmd := range cmds {
			if e.Rune() != key {
				continue
			}
			fn, err := gomu.command.getFn(cmd)
			if err != nil {
				logError(err)
				return e
			}
			fn()
		}

		return e
	})

	// fix transparent background issue
	application.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	go populateAudioLength(gomu.playlist.GetRoot())
	// main loop
	if err := application.SetRoot(gomu.pages, true).SetFocus(gomu.playlist).Run(); err != nil {
		logError(err)
	}

}

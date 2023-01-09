// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"

	"github.com/issadarkthing/gomu/anko"
	"github.com/issadarkthing/gomu/hook"
	"github.com/issadarkthing/gomu/player"
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

// Default values for command line arguments.
// TODO: change to $XDG_* variables
const (
	configPath     = "~/.config/gomu/config"
	cacheQueuePath = "~/.local/share/gomu/queue.cache"
	musicPath      = "~/music" //by default this is uppercase
)

// Args is the augs for gomu executable
type Args struct {
	config  *string
	empty   *bool
	music   *string
	version *bool
}

func getArgs() Args {
	configFlag := flag.String("config", configPath, "Specify config file")
	emptyFlag := flag.Bool("empty", false, "Open gomu with empty queue. Does not override previous queue")
	musicFlag := flag.String("music", musicPath, "Specify music directory")
	versionFlag := flag.Bool("version", false, "Print gomu version")
	flag.Parse()
	return Args{
		config:  configFlag,
		empty:   emptyFlag,
		music:   musicFlag,
		version: versionFlag,
	}
}

// built-in functions
func defineBuiltins() {
	gomu.anko.DefineGlobal("debug_popup", debugPopup)
	gomu.anko.DefineGlobal("info_popup", infoPopup)
	gomu.anko.DefineGlobal("input_popup", inputPopup)
	gomu.anko.DefineGlobal("show_popup", defaultTimedPopup)
	gomu.anko.DefineGlobal("search_popup", searchPopup)
	gomu.anko.DefineGlobal("shell", shell)
}

func defineInternals() {
	playlist, _ := gomu.anko.NewModule("Playlist")
	playlist.Define("get_focused", gomu.playlist.getCurrentFile)
	playlist.Define("focus", func(filepath string) {

		root := gomu.playlist.GetRoot()
		root.Walk(func(node, _ *tview.TreeNode) bool {

			if node.GetReference().(*player.AudioFile).Path() == filepath {
				gomu.playlist.setHighlight(node)
				return false
			}

			return true
		})
	})

	queue, _ := gomu.anko.NewModule("Queue")
	queue.Define("get_focused", func() *player.AudioFile {
		index := gomu.queue.GetCurrentItem()
		if index < 0 || index > len(gomu.queue.items)-1 {
			return nil
		}
		item := gomu.queue.items[index]
		return item
	})

	player, _ := gomu.anko.NewModule("Player")
	player.Define("current_audio", gomu.player.GetCurrentSong)
}

func setupHooks(hook *hook.EventHook, anko *anko.Anko) {

	events := []string{
		"enter",
		"new_song",
		"skip",
		"play",
		"pause",
		"exit",
	}

	for _, event := range events {
		name := event
		hook.AddHook(name, func() {
			src := fmt.Sprintf(`Event.run_hooks("%s")`, name)
			_, err := anko.Execute(src)
			if err != nil {
				err = tracerr.Errorf("error execute hook: %w", err)
				logError(err)
			}
		})
	}
}

// loadModules executes helper modules and default config that should only be
// executed once
func loadModules(env *anko.Anko) error {

	const listModule = `
module List {

	func collect(l, f) {
		result = []
		for x in l {
			result += f(x)
		}
		return result
	}

	func filter(l, f) {
		result = []
		for x in l {
			if f(x) {
				result += x
			}
		}
		return result
	}

	func reduce(l, f, acc) {
		for x in l {
			acc = f(acc, x)
		}
		return acc
	}
}
`
	const eventModule = `
module Event {
	events = {}

	func add_hook(name, f) {
		hooks = events[name]

		if hooks == nil {
			events[name] = [f]
			return
		}

		hooks += f
		events[name] = hooks
	}

	func run_hooks(name) {
		hooks = events[name]

		if hooks == nil {
			return
		}

		for hook in hooks {
			hook()
		}
	}
}
`

	const keybindModule = `
module Keybinds {
	global = {}
	playlist = {}
	queue = {}

	func def_g(kb, f) {
		global[kb] = f
	}

	func def_p(kb, f) {
		playlist[kb] = f
	}

	func def_q(kb, f) {
		queue[kb] = f
	}
}
`
	_, err := env.Execute(eventModule + listModule + keybindModule)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// executes user config with default config is executed first in order to apply
// default values
func execConfig(config string) error {

	const defaultConfig = `

module General {
	# confirmation popup to add the whole playlist to the queue
	confirm_bulk_add    = true
	confirm_on_exit     = true
	queue_loop          = false
	load_prev_queue     = true
	popup_timeout       = "5s"
	sort_by_mtime       = false
	# change this to directory that contains mp3 files
	music_dir           = "~/Music"
	# url history of downloaded audio will be saved here
	history_path        = "~/.local/share/gomu/urls"
	# some of the terminal supports unicode character
	# you can set this to true to enable emojis
	use_emoji           = true
	# initial volume when gomu starts up
	volume              = 80
	# if you experiencing error using this invidious instance, you can change it
	# to another instance from this list:
	# https://github.com/iv-org/documentation/blob/master/Invidious-Instances.md
	invidious_instance  = "https://vid.puffyan.us"
	# Prefered language for lyrics to be displayed, if not available, english version
	# will be displayed.
	# Available tags: en,el,ko,es,th,vi,zh-Hans,zh-Hant,zh-CN and can be separated with comma.
	# find more tags: youtube-dl --skip-download --list-subs "url"
	lang_lyric          = "en"
	# When save tag, could rename the file by tag info: artist-songname-album
	rename_bytag        = false
}

module Emoji {
	# default emoji here is using awesome-terminal-fonts
	# you can change these to your liking
	playlist     = ""
	file         = ""
	loop         = "ﯩ"
	noloop       = ""
}

module Color {
	# you may choose colors by pressing 'c'
	accent            = "darkcyan"
	background        = "none"
	foreground        = "white"
	popup             = "black"

	playlist_directory = "darkcyan"
	playlist_highlight = "darkcyan"

	queue_highlight    = "darkcyan"

	now_playing_title = "darkgreen"
	subtitle          = "darkgoldenrod"
}

# you can get the syntax highlighting for this language here:
# https://github.com/mattn/anko/tree/master/misc/vim
# vim: ft=anko
`

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
	_, err = gomu.anko.Execute(defaultConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// execute user config
	_, err = gomu.anko.Execute(string(content))
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// Sets the layout of the application
func layout(gomu *Gomu) *tview.Flex {
	flex := tview.NewFlex().
		AddItem(gomu.playlist, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(gomu.queue, 0, 5, false).
			AddItem(gomu.playingBar, 9, 0, false), 0, 2, false)

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
	gomu.command.defineCommands()
	defineBuiltins()

	err := loadModules(gomu.anko)
	if err != nil {
		die(err)
	}

	err = execConfig(expandFilePath(*args.config))
	if err != nil {
		die(err)
	}

	setupHooks(gomu.hook, gomu.anko)

	gomu.hook.RunHooks("enter")
	gomu.args = args
	gomu.colors = newColor()

	// override default border
	// change double line border to one line border when focused
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
	tview.Styles.PrimitiveBackgroundColor = gomu.colors.popup

	gomu.initPanels(application, args)
	defineInternals()

	gomu.player.SetSongStart(func(audio player.Audio) {

		duration, err := getTagLength(audio.Path())
		if err != nil || duration == 0 {
			duration, err = player.GetLength(audio.Path())
			if err != nil {
				logError(err)
				return
			}
		}

		audioFile := audio.(*player.AudioFile)

		gomu.playingBar.newProgress(audioFile, int(duration.Seconds()))

		name := audio.Name()
		var description string

		if len(gomu.playingBar.subtitles) == 0 {
			description = name
		} else {
			lang := gomu.playingBar.subtitle.LangExt

			description = fmt.Sprintf("%s \n\n %s lyric loaded", name, lang)
		}

		defaultTimedPopup(" Now Playing ", description)

		go func() {
			err := gomu.playingBar.run()
			if err != nil {
				logError(err)
			}
		}()

	})

	gomu.player.SetSongFinish(func(currAudio player.Audio) {

		gomu.playingBar.subtitles = nil
		var mu sync.Mutex
		mu.Lock()
		gomu.playingBar.subtitle = nil
		mu.Unlock()
		if gomu.queue.isLoop {
			_, err = gomu.queue.enqueue(currAudio.(*player.AudioFile))
			if err != nil {
				logError(err)
			}
		}

		if len(gomu.queue.items) > 0 {
			err := gomu.queue.playQueue()
			if err != nil {
				logError(err)
			}
		} else {
			gomu.playingBar.setDefault()
		}
	})

	flex := layout(gomu)
	gomu.pages.AddPage("main", flex, true, true)

	// sets the first focused panel
	gomu.setFocusPanel(gomu.playlist)
	gomu.prevPanel = gomu.playlist

	gomu.playingBar.setDefault()

	gomu.queue.isLoop = gomu.anko.GetBool("General.queue_loop")

	loadQueue := gomu.anko.GetBool("General.load_prev_queue")

	if !*args.empty && loadQueue {
		// load saved queue from previous session
		if err := gomu.queue.loadQueue(); err != nil {
			logError(err)
		}
	}

	if len(gomu.queue.items) > 0 {
		if err := gomu.queue.playQueue(); err != nil {
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
		'm': "repl",
		'T': "switch_lyric",
		'c': "show_colors",
	}

	for key, cmdName := range cmds {
		src := fmt.Sprintf(`Keybinds.def_g("%c", %s)`, key, cmdName)
		gomu.anko.Execute(src)
	}

	// global keybindings are handled here
	application.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		if gomu.pages.HasPage("repl-input-popup") {
			return e
		}

		if gomu.pages.HasPage("tag-editor-input-popup") {
			return e
		}

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

		if gomu.anko.KeybindExists("global", e) {

			err := gomu.anko.ExecKeybind("global", e)
			if err != nil {
				errorPopup(err)
			}

			return nil
		}

		return e
	})

	// fix transparent background issue
	gomu.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	init := false
	gomu.app.SetAfterDrawFunc(func(_ tcell.Screen) {
		if !init && len(gomu.queue.items) == 0 {
			gomu.playingBar.setDefault()
			init = true
		}
	})

	gomu.app.SetRoot(gomu.pages, true).SetFocus(gomu.playlist)

	// main loop
	if err := gomu.app.Run(); err != nil {
		die(err)
	}

	gomu.hook.RunHooks("exit")
}

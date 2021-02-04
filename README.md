
# Gomu (Go Music Player) [![Go Report Card](https://goreportcard.com/badge/github.com/issadarkthing/gomu)](https://goreportcard.com/report/github.com/issadarkthing/gomu) [![Build Status](https://travis-ci.com/issadarkthing/gomu.svg?branch=master)](https://travis-ci.com/issadarkthing/gomu)
Gomu is a Terminal User Interface **TUI** music player to play mp3 files from your local machine. 

![demo](/gomu.gif)

## Features
- lightweight
- simple
- fast
- show audio files as tree
- queue cache
- [vim](https://github.com/vim/vim) keybindings
- [fzf](https://github.com/junegunn/fzf) integration
- [youtube-dl](https://github.com/ytdl-org/youtube-dl) integration
- audio file management
- customizable

## Dependencies
If you are using ubuntu, you need to install alsa and required dependencies
```sh
$ sudo apt install libasound2-dev go
```
Optional dependencies can be installed by this command
```sh
$ sudo apt install youtube-dl fzf fonts-noto
```

## Installation

```sh
$ go get -u github.com/issadarkthing/gomu
```

For arch users, you can install from the AUR

using [yay](https://github.com/Jguer/yay):
```sh
$ yay -S gomu
```
using [aura](https://github.com/fosskers/aura):
```sh
$ sudo aura -A gomu
```


## Configuration
By default, gomu will look for audio files in `~/music` directory. If you wish to change to your desired location, edit `~/.config/gomu/config` file
and change `music_dir: path/to/your/musicDir`. 

Sample config file:

```
color:
  accent:            "#008B8B"
  background:        none
  foreground:        "#FFFFFF"
  now_playing_title: "#017702"
  playlist:          "#008B8B"
  popup:             "#0A0F14"

general:
  confirm_bulk_add:  true
  confirm_on_exit:   true
  load_prev_queue:   true
  music_dir:         ~/music
  popup_timeout:     5s
  volume:            100
  emoji:             true
  fzf:               false
```

## Fzf
Eventhough gomu can use [fzf](https://github.com/junegunn/fzf) as its finder but it is recommended to use built-in
finder. This is due to the bug which may cause the application to hang up
if fzf is being used for a long period of time (not everytime). As of `v1.5.0`,
the default built-in finder will be used instead of fzf. To override this behaviour,
edit this line `fzf: false` to change it into `true` in `~/.config/gomu/config`.


## Keybindings
Each panel has it's own additional keybinding. To view the available keybinding for the specific panel use `?`

| Key             |            Description |
|:----------------|-----------------------:|
| j               |                   down |
| k               |                     up |
| tab             |           change panel |
| space           |      toggle play/pause |
| esc             |            close popup |
| n               |                   skip |
| q               |                   quit |
| l (lowercase L) |      add song to queue |
| L               |  add playlist to queue |
| h               | close node in playlist |
| d               |      remove from queue |
| D               |        delete playlist |
| +               |              volume up |
| -               |            volume down |
| Y               |         download audio |
| a               |        create playlist |
| ?               |            toggle help |



## Project Background
I just wanted to implement my own music player with a programming language i'm currently learning ([Go](https://golang.org/)). Gomu might not be stable as it in constant development. For now, it can fulfill basic music player functions such as:
- add and delete songs from queue
- create playlists
- skip
- play
- pause 

Seeking and more advanced stuff has not yet been implemented; feel free to contribute :)

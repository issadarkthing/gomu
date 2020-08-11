
# Gomu (Go Music Player) [![Go Report Card](https://goreportcard.com/badge/github.com/issadarkthing/gomu)](https://goreportcard.com/report/github.com/issadarkthing/gomu) [![Build Status](https://travis-ci.com/issadarkthing/gomu.svg?branch=master)](https://travis-ci.com/issadarkthing/gomu)
Gomu is a Terminal User Interface **TUI** music player to play mp3 files from your local machine. 

![demo](/gomu.gif)

## Features
- lightweight
- simple
- fast
- show audio files as tree
- queue cache
- vim keybindings
- fzf integration
- youtube-dl integration
- audio file management
- customizeable

## Dependencies
If you are on ubuntu, you need to install alsa as required dependencies
```sh
$ sudo apt install libasound2-dev go
```
Optional dependencies can be installed by this command
```sh
$ sudo apt install youtube-dl fzf
```

## Installation
```sh
go get -u github.com/issadarkthing/gomu
```

## Configuration
By default, gomu will look for audio files in `~/music` directory. If you wish to change to your desired location, edit `~/.config/gomu/config` file
and change `music_dir: path/to/your/musicDir`. Example of the config file will look like:

```
confirm_on_exit:  true
music_dir:        ~/music
confirm_bulk_add: true
popup_timeout:    5
```

## Keybindings

|  Key   |       Description      |
|--------|------------------------|
| j      | down                   |
| k      | up                     |
| tab    | change panel           |
| space  | toggle play/pause      |
| esc    | close popup            |
| n      | skip                   |
| q      | quit                   |
| l      | add song to queue      |
| L      | add playlist to queue  |
| h      | close node in playlist |
| d      | remove from queue      |
| D      | delete playlist        |
| +      | volume up              |
| -      | volume down            |
| Y      | download audio         |
| a      | create playlist        |
| ?      | toggle help            |



## Project Background
I just want to implement my own music player with a programming language im currently learning [Go](https://golang.org/). Gomu might not be stable as it in constant development. For now, it can do basic music player can do like adding, deleting songs from queue, skip, play, pause but not seeking or more advanced stuff; feel free to contribute :)

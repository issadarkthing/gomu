
## Gomu (Go Music Player) 
[![Go Report Card](https://goreportcard.com/badge/github.com/issadarkthing/gomu)](https://goreportcard.com/report/github.com/issadarkthing/gomu) [![Build Status](https://travis-ci.com/issadarkthing/gomu.svg?branch=master)](https://travis-ci.com/issadarkthing/gomu)
<a href="https://www.buymeacoffee.com/raziman" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="41" width="174"></a>

Gomu is intuitive, powerful CLI music player. It has embedded scripting language
and event hook to enable user to customize their config extensively.

![gomu](https://user-images.githubusercontent.com/50593529/107107772-37fdc000-686e-11eb-8c0f-c7d7f43f3c80.png)

### Features
- lightweight
- simple
- fast
- show audio files as tree
- queue cache
- [vim](https://github.com/vim/vim) keybindings
- [youtube-dl](https://github.com/ytdl-org/youtube-dl) integration
- audio file management
- customizable
- find music from youtube
- scriptable config
- download lyric
- id3v2 tag editor

### Dependencies
If you are using ubuntu, you need to install alsa and required dependencies
```sh
sudo apt install libasound2-dev go
```
Optional dependencies can be installed by this command
```sh
sudo apt install youtube-dl
```

### Installation

Using the latest vesrion of go:

```sh
go install github.com/issadarkthing/gomu@latest
```

Older versions of go can use

```sh
go get -u github.com/issadarkthing/gomu
```

For arch users, you can install from the AUR

using [yay](https://github.com/Jguer/yay):
```sh
yay -S gomu
```
using [aura](https://github.com/fosskers/aura):
```sh
sudo aura -A gomu
```


### Configuration
By default, gomu will look for audio files in `~/music` directory. If you wish to change to your desired location, edit `~/.config/gomu/config` file
and change `music_dir = path/to/your/musicDir`. 


### Keybindings
Each panel has it's own additional keybinding. To view the available keybinding for the specific panel use `?`

| Key (General)   |                     Description |
|:----------------|--------------------------------:|
| tab             |                    change panel |
| space           |               toggle play/pause |
| esc             |                     close popup |
| n               |                            skip |
| q               |                            quit |
| +               |                       volume up |
| -               |                     volume down |
| f/F             |           forward 10/60 seconds |
| b/B             |            rewind 10/60 seconds |
| ?               |                     toggle help |
| m               |                       open repl |
| T               |                   switch lyrics |
| c               |                     show colors |


| Key (Playlist)  |                     Description |
|:----------------|--------------------------------:|
| j               |                            down |
| k               |                              up |
| h               |          close node in playlist |
| a               |                 create playlist |
| l (lowercase L) |               add song to queue |
| L               |           add playlist to queue |
| d               |    delete file from filesystemd |
| D               | delete playlist from filesystem |
| Y               |                  download audio |
| r               |                         refresh |
| R               |                          rename |
| y/p             |                 yank/paste file |
| /               |                find in playlist |
| s               |       search audio from youtube |
| t               |                   edit mp3 tags |
| 1/2             |         find lyric if available |

| Key (Queue)     |                     Description |
|:----------------|--------------------------------:|
| j               |                            down |
| k               |                              up |
| l (lowercase L) |              play selected song |
| d               |               remove from queue |
| D               |                 delete playlist |
| z               |                     toggle loop |
| s               |                         shuffle |
| /               |                   find in queue |
| t               | lyric delay increase 0.5 second |
| r               | lyric delay decrease 0.5 second |

### Scripting

Gomu uses [anko](https://github.com/mattn/anko) as its scripting language. You can read
more about scripting at our [wiki](https://github.com/issadarkthing/gomu/wiki)

``` go

Keybinds.def_g("ctrl_x", func() {
    out, err = shell(`echo "hello world"`)
    if err != nil {
        debug_popup("an error occured")
    }
    info_popup(out)
})

```

### Project Background
I just wanted to implement my own music player with a programming language i'm currently learning ([Go](https://golang.org/)). Gomu might not be stable as it in constant development. For now, it can fulfill basic music player functions such as:
- add and delete songs from queue
- create playlists
- skip
- play
- pause 
- forward and rewind

### Similar Projects
- [termusic](https://github.com/tramhao/termusic) (Written in rust and well maintained)

### Album Photo
For songs downloaded by Gomu, the thumbnail will be embeded as Album cover. If you're not satisfied with the cover, you can edit it with kid3 and attach an image as album cover. Jpeg is tested, but other formats should work as well.

### Donation
Hi! If you guys think the project is cool, you can buy me a coffee ;)

<a href="https://www.buymeacoffee.com/raziman" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="50" width="180"></a>

Seeking and more advanced stuff has not yet been implemented; feel free to contribute.

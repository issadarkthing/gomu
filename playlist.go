// Copyright (C) 2020  Raziman

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

type AudioFile struct {
	Name        string
	Path        string
	IsAudioFile bool
	Length      time.Duration
	Parent      *tview.TreeNode
}

type Playlist struct {
	*tview.TreeView
	prevNode *tview.TreeNode
}

func (p *Playlist) Help() []string {

	return []string{
		"j      down",
		"k      up",
		"h      close node",
		"l      add song to queue",
		"L      add playlist to queue",
		"d      delete file from filesystem",
		"D      delete playlist from filesystem",
		"Y      download audio",
		"r      refresh",
	}

}

func NewPlaylist() *Playlist {

	rootDir, err := filepath.Abs(expandTilde(viper.GetString("music_dir")))

	if err != nil {
		log.Println(err)
	}

	root := tview.NewTreeNode(path.Base(rootDir)).
		SetColor(gomu.AccentColor)

	tree := tview.NewTreeView().SetRoot(root)

	playlist := &Playlist{
		TreeView: tree,
	}

	rootAudioFile := &AudioFile{
		Name: root.GetText(),
		Path: rootDir,
	}

	root.SetReference(rootAudioFile)

	playlist.
		SetTitle(" Playlist ").
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft)

	populate(root, rootDir)

	var firstChild *tview.TreeNode

	if len(root.GetChildren()) == 0 {
		firstChild = root
	} else {
		firstChild = root.GetChildren()[0]
	}

	playlist.SetHighlight(firstChild)

	playlist.SetChangedFunc(func(node *tview.TreeNode) {
		playlist.SetHighlight(node)
	})

	playlist.SetSelectedFunc(func(node *tview.TreeNode) {
		node.SetExpanded(!node.IsExpanded())
	})

	playlist.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		currNode := playlist.GetCurrentNode()

		audioFile := currNode.GetReference().(*AudioFile)

		switch e.Rune() {

		case 'a':

			name, _ := gomu.Pages.GetFrontPage()

			if name != "mkdir-popup" {
				CreatePlaylistPopup()
			}

		case 'D':

			var selectedDir *AudioFile

			// gets the parent dir if current focused node is not a dir
			if audioFile.IsAudioFile {
				selectedDir = audioFile.Parent.GetReference().(*AudioFile)
			} else {
				selectedDir = audioFile
			}

			confirmationPopup(
				"Are you sure to delete this directory?", func(_ int, buttonName string) {

					if buttonName == "no" {
						gomu.Pages.RemovePage("confirmation-popup")
						gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
						return
					}

					err := os.RemoveAll(selectedDir.Path)

					if err != nil {
						timedPopup(
							" Error ",
							"Unable to delete dir "+selectedDir.Name, getPopupTimeout(), 0, 0)
					} else {
						timedPopup(
							" Success ",
							selectedDir.Name+"\nhas been deleted successfully", getPopupTimeout(), 0, 0)

						playlist.Refresh()
					}

					gomu.Pages.RemovePage("confirmation-popup")
					gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))

				})

		case 'd':

			// prevent from deleting a directory
			if !audioFile.IsAudioFile {
				return e
			}

			confirmationPopup(
				"Are you sure to delete this audio file?", func(_ int, buttonName string) {

					if buttonName == "no" {
						gomu.Pages.RemovePage("confirmation-popup")
						gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
						return
					}

					err := os.Remove(audioFile.Path)

					if err != nil {
						timedPopup(
							" Error ", "Unable to delete "+audioFile.Name, getPopupTimeout(), 0, 0)
					} else {
						timedPopup(
							" Success ",
							audioFile.Name+"\nhas been deleted successfully", getPopupTimeout(), 0, 0)

						playlist.Refresh()
					}

					gomu.Pages.RemovePage("confirmation-popup")
					gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
				})

		case 'Y':

			if gomu.Pages.HasPage("download-popup") {
				gomu.Pages.RemovePage("download-popup")
				return e
			}

			// this ensures it downloads to
			// the correct dir
			if audioFile.IsAudioFile {
				downloadMusicPopup(audioFile.Parent)
			} else {
				downloadMusicPopup(currNode)
			}

		case 'l':

			if audioFile.IsAudioFile {
				gomu.Queue.Enqueue(audioFile)
			} else {
				currNode.SetExpanded(true)
			}

		case 'h':

			// if closing node with no children
			// close the node's parent
			// remove the color of the node

			if audioFile.IsAudioFile {
				parent := audioFile.Parent

				playlist.SetHighlight(parent)

				parent.SetExpanded(false)
			}

			currNode.Collapse()

		case 'L':

			if !viper.GetBool("confirm_bulk_add") {
				playlist.AddAllToQueue(playlist.GetCurrentNode())
				return e
			}

			confirmationPopup(
				"Are you sure to add this whole directory into queue?",
				func(_ int, label string) {

					if label == "yes" {
						playlist.AddAllToQueue(playlist.GetCurrentNode())
					}

					gomu.Pages.RemovePage("confirmation-popup")
					gomu.App.SetFocus(playlist)

				})

		case 'r':

			playlist.Refresh()

		}

		return e
	})

	return playlist

}

// Add songs and their directories in Playlist panel
func populate(root *tview.TreeNode, rootPath string) {

	files, err := ioutil.ReadDir(rootPath)

	if err != nil {
		log.Println(err)
	}

	for _, file := range files {

		path := filepath.Join(rootPath, file.Name())
		f, err := os.Open(path)

		if err != nil {
			log.Println(err)
			continue
		}

		defer f.Close()

		songName := GetName(file.Name())
		child := tview.NewTreeNode(songName)

		if !file.IsDir() {

			filetype, err := GetFileContentType(f)

			if err != nil {
				log.Println(err)
				continue
			}

			// skip if not mp3 file
			if filetype != "mpeg" {
				continue
			}

			audioLength, err := GetLength(path)

			if err != nil {
				log.Println(err)
				continue
			}

			audioFile := &AudioFile{
				Name:        songName,
				Path:        path,
				IsAudioFile: true,
				Length:      audioLength,
				Parent:      root,
			}

			child.SetReference(audioFile)
			root.AddChild(child)

		}

		if file.IsDir() {

			audioFile := &AudioFile{
				Name:        songName,
				Path:        path,
				IsAudioFile: false,
				Length:      0,
				Parent:      root,
			}
			child.SetReference(audioFile)
			child.SetColor(gomu.AccentColor)
			root.AddChild(child)
			populate(child, path)

		}

	}

}

// Bulk add a playlist to queue
func (p *Playlist) AddAllToQueue(root *tview.TreeNode) {

	var childrens []*tview.TreeNode

	childrens = root.GetChildren()

	// gets the parent if the highlighted item is a file
	if root.GetReference().(*AudioFile).IsAudioFile {
		childrens = root.GetReference().(*AudioFile).Parent.GetChildren()
	}

	for _, v := range childrens {
		currNode := v.GetReference().(*AudioFile)

		gomu.Queue.Enqueue(currNode)
	}

}

// Triggers fzf to find in current directory
func (p *Playlist) FuzzySearch() {

	_, err := exec.LookPath("fzf")

	if err != nil {
		timedPopup(" Error ", "FZF not found in your $PATH", getPopupTimeout(), 0, 0)
		log.Println(err)
		return
	}

	rootPath := p.GetRoot().GetReference().(*AudioFile).Path

	cmd := exec.Command("fzf", "--multi")
	stdin, err := cmd.StdinPipe()

	go func(c io.WriteCloser) {
		filepath.Walk(rootPath, func(path string, f os.FileInfo, err error) error {

			io.WriteString(c, path)
			return err
		})
	}(stdin)

}

// Refresh the playlist and read the whole root music dir
func (p *Playlist) Refresh() {

	root := gomu.Playlist.GetRoot()

	prevFileName := gomu.Playlist.GetCurrentNode().GetText()

	root.ClearChildren()

	populate(root, root.GetReference().(*AudioFile).Path)

	root.Walk(func(node, parent *tview.TreeNode) bool {

		// to preserve previously highlighted node
		if node.GetText() == prevFileName {
			p.SetHighlight(node)
			return false
		}

		return true
	})

}

// Adds child while setting reference to audio file
func (p *Playlist) AddSongToPlaylist(audioPath string, selPlaylist *tview.TreeNode) error {

	f, err := os.Open(audioPath)

	if err != nil {
		return err
	}

	defer f.Close()

	node := tview.NewTreeNode(GetName(audioPath))

	audioLength, err := GetLength(audioPath)

	if err != nil {
		return err
	}

	audioFile := &AudioFile{
		Name:        path.Base(audioPath),
		Path:        audioPath,
		IsAudioFile: true,
		Length:      audioLength,
		Parent:      selPlaylist,
	}

	node.SetReference(audioFile)
	selPlaylist.AddChild(node)
	gomu.App.Draw()

	return nil

}

func (p *Playlist) GetAudioFiles() []*AudioFile {

	root := p.GetRoot()

	audioFiles := []*AudioFile{}

	root.Walk(func(node, _ *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)
		audioFiles = append(audioFiles, audioFile)

		return true
	})

	return audioFiles
}

// Creates a directory under selected node, returns error if playlist exists
func (p *Playlist) CreatePlaylist(name string) error {

	selectedNode := p.GetCurrentNode()

	parentNode := selectedNode.GetReference().(*AudioFile).Parent

	// if the current node is the root
	// sets the parent to itself
	if parentNode == nil {
		parentNode = selectedNode
	}

	audioFile := parentNode.GetReference().(*AudioFile)

	err := os.Mkdir(path.Join(audioFile.Path, name), 0744)

	if err != nil {
		return err
	}

	p.Refresh()

	return nil

}

// This is used to replace default behaviour of SetCurrentNode which
// adds color highlight attributes
func (p *Playlist) SetHighlight(currNode *tview.TreeNode) {

	if p.prevNode != nil {
		p.prevNode.SetColor(gomu.TextColor)
	}
	currNode.SetColor(gomu.AccentColor)
	p.SetCurrentNode(currNode)

	if currNode.GetReference().(*AudioFile).IsAudioFile {
		p.prevNode = currNode
	}

}

// Traverses the playlist and finds the AudioFile struct
// audioName must be hashed with sha1 first
func (p *Playlist) FindAudioFile(audioName string) *AudioFile {

	root := p.GetRoot()

	if root == nil {
		return nil
	}

	var selNode *AudioFile

	root.Walk(func(node, _ *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)

		hashed := Sha1Hex(GetName(audioFile.Name))

		if hashed == audioName {
			selNode = audioFile
			return false
		}

		return true
	})

	return selNode

}

// download audio from youtube audio and adds the song to the selected playlist
func Ytdl(url string, selPlaylist *tview.TreeNode) (error, chan error) {

	// lookup if youtube-dl exists
	_, err := exec.LookPath("youtube-dl")

	if err != nil {
		timedPopup(" Error ", "youtube-dl is not in your $PATH", getPopupTimeout(), 0, 0)
		return err, nil
	}

	dir := viper.GetString("music_dir")

	selAudioFile := selPlaylist.GetReference().(*AudioFile)
	selPlaylistName := selAudioFile.Name

	timedPopup(" Ytdl ", "Downloading", getPopupTimeout(), 0, 0)

	// specify the output path for ytdl
	outputDir := fmt.Sprintf(
		"%s/%s/%%(title)s.%%(ext)s",
		dir,
		selPlaylistName)

	args := []string{
		"--extract-audio",
		"--audio-format",
		"mp3",
		"--output",
		outputDir,
		url,
	}

	cmd := exec.Command("youtube-dl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	e := make(chan error)

	go func() {

		err := cmd.Run()

		if err != nil {
			timedPopup(" Error ", "Error running youtube-dl", getPopupTimeout(), 0, 0)
			log.Println(err)
			e <- err
			return
		}

		playlistPath := path.Join(expandTilde(dir), selPlaylistName)

		audioPath := extractFilePath(stdout.Bytes(), playlistPath)

		if err != nil {
			log.Println(err)
			e <- err
		}

		err = gomu.Playlist.AddSongToPlaylist(audioPath, selPlaylist)

		if err != nil {
			log.Println(err)
			e <- err
		}

		downloadFinishedMessage := fmt.Sprintf("Finished downloading\n%s",
			path.Base(audioPath))

		timedPopup(
			" Ytdl ",
			downloadFinishedMessage,
			getPopupTimeout(), 0, 0)

		gomu.App.Draw()

		e <- nil

	}()

	return nil, e

}

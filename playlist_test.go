package main

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/issadarkthing/gomu/player"
	"github.com/rivo/tview"
)

// Prepares for test
func prepareTest() *Gomu {

	gomu := newGomu()
	gomu.player = player.NewPlayer(0, "beep")
	gomu.queue = newQueue()
	gomu.playlist = &Playlist{
		TreeView: tview.NewTreeView(),
	}
	gomu.app = tview.NewApplication()

	err := execConfig(expandFilePath(testConfigPath))
	if err != nil {
		panic(err)
	}

	gomu.colors = newColor()

	rootDir, err := filepath.Abs("./test")
	if err != nil {
		panic(err)
	}

	root := tview.NewTreeNode("music")
	rootAudioFile := new(player.AudioFile)
	rootAudioFile.SetName(root.GetText())
	rootAudioFile.SetPath(rootDir)

	root.SetReference(rootAudioFile)
	populate(root, rootDir, false)
	gomu.playlist.SetRoot(root)

	return gomu
}

func TestPopulate(t *testing.T) {

	gomu = newGomu()
	err := execConfig(expandFilePath(testConfigPath))
	if err != nil {
		t.Error(err)
	}
	gomu.colors = newColor()

	rootDir, err := filepath.Abs("./test")

	if err != nil {
		panic(err)
	}

	expected := 0
	walkFn := func(path string, info os.FileInfo, err error) error {

		if info.IsDir() {
			expected++
			return nil
		}

		f, e := os.Open(path)
		if e != nil {
			return e
		}

		defer f.Close()

		expected++

		return nil
	}

	// calculate the amount of mp3 files and directories
	filepath.Walk(rootDir, walkFn)

	root := tview.NewTreeNode(path.Base(rootDir))

	rootAudioFile := new(player.AudioFile)
	rootAudioFile.SetName("Music")
	rootAudioFile.SetIsAudioFile(false)

	populate(root, rootDir, false)
	gotItems := 0
	root.Walk(func(node, _ *tview.TreeNode) bool {
		gotItems++
		return true
	})

	// ignore config, config.test and arbitrary_file.txt
	gotItems += 3

	if gotItems != expected {
		t.Errorf("Invalid amount of file; expected %d got %d", expected, gotItems)
	}

}

func TestAddAllToQueue(t *testing.T) {

	gomu = prepareTest()
	var songs []*tview.TreeNode

	gomu.playlist.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {

		if node.GetReference().(*player.AudioFile).Name() == "rap" {
			gomu.playlist.addAllToQueue(node)
		}

		return true
	})

	queue := gomu.queue.getItems()

	for i, song := range songs {

		audioFile := song.GetReference().(*player.AudioFile)

		// strips the path of the song in the queue
		s := filepath.Base(queue[i])

		if audioFile.Name() != s {
			t.Errorf("Expected \"%s\", got \"%s\"", audioFile.Name(), s)
		}

	}

}

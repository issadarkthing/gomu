package main

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/rivo/tview"
)

func preparePlaylist() *Gomu {

	gomu := NewGomu()
	gomu.Player = &Player{}
	gomu.Queue = NewQueue()
	gomu.Playlist = &Playlist{
		tview.NewTreeView(),
		nil,
	}
	gomu.App = tview.NewApplication()

	rootDir, err := filepath.Abs("./music")
	if err != nil {
		panic(err)
	}

	root := tview.NewTreeNode("music")
	rootAudioFile := &AudioFile{
		Name:        root.GetText(),
		Path:        rootDir,
		IsAudioFile: false,
		Parent:      nil,
	}

	root.SetReference(rootAudioFile)
	populate(root, rootDir)
	gomu.Playlist.SetRoot(root)

	return gomu
}

func TestPopulate(t *testing.T) {

	gomu = NewGomu()

	rootDir, err := filepath.Abs("./music")

	if err != nil {
		panic(err)
	}

	items := 0

	// calculate the amount of mp3 files
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {

		if info.IsDir() {
			items++
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}

		defer f.Close()


		fileType, err := GetFileContentType(f)

		if err != nil {
			return err
		}

		if fileType == "mpeg" {
			items++
		}

		return nil
	})

	root := tview.NewTreeNode(path.Base(rootDir))

	populate(root, rootDir)

	gotItems := 0
	root.Walk(func(node, parent *tview.TreeNode) bool {
		gotItems++
		return true
	})

	if gotItems != items {
		t.Error("populate() does not return correct amount of file")
	}

}

func TestAddToQueue(t *testing.T) {

	gomu = preparePlaylist()

	var selNode []*AudioFile

	gomu.Playlist.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)

		if len(selNode) < 2 && audioFile.IsAudioFile {
			selNode = append(selNode, audioFile)
			return false
		}

		return true
	})

	for _, v := range selNode {
		gomu.Playlist.addToQueue(v)
	}

	queueLen := gomu.Queue.GetItemCount()

	if queueLen != 1 {
		t.Errorf("Invalid count in queue; expected %d, got %d", 1, queueLen)
	}
	
}



func TestAddAllToQueue(t *testing.T) {

	gomu = preparePlaylist()

	var selNode []*AudioFile

	gomu.Playlist.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)

		if audioFile.IsAudioFile {
			selNode = append(selNode, audioFile)
			return false
		}

		return true
	})

	for _, v := range selNode {
		gomu.Playlist.addToQueue(v)
	}

	queueLen := gomu.Queue.GetItemCount()

	// we subtract selNode by one because the first element in
	// the queue will be poppped anyway
	if len(selNode)-1 != queueLen {
		t.Errorf("Invalid count in queue; expected %d, got %d", len(selNode)-1, queueLen)
	}
	
}

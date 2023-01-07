package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/issadarkthing/gomu/anko"
	"github.com/issadarkthing/gomu/hook"
	"github.com/stretchr/testify/assert"
)

func TestGetArgs(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	// Test default case
	ar := getArgs()
	assert.Equal(t, *ar.config, "~/.config/gomu/config")
	assert.Equal(t, *ar.empty, false)
	assert.Equal(t, *ar.music, "~/music")
	assert.Equal(t, *ar.version, false)

	// Test setting config flag
	testConfig := filepath.Join(cfgDir, ".tmp", "gomu")
	_, err = os.Stat(testConfig)
	if os.IsNotExist(err) {
		os.MkdirAll(testConfig, 0755)
	}
	defer os.RemoveAll(testConfig)
	//create a temporary config file
	tmpCfgf, err := os.CreateTemp(testConfig, "config")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	os.Args = []string{"cmd", "-config", tmpCfgf.Name()}
	ar = getArgs()
	assert.Equal(t, *ar.config, tmpCfgf.Name())

	// Test -empty flag
	os.Args = []string{"cmd", "-empty"}
	ar = getArgs()
	assert.Equal(t, *ar.empty, true)
	assert.Zero(t, gomu.queue) // not sure if that's correct

	// Test setting music flag
	testMusic := filepath.Join(home, ".tmp", "gomu")
	_, err = os.Stat(testMusic)
	if os.IsNotExist(err) {
		os.MkdirAll(testMusic, 0755)
	}
	defer os.RemoveAll(testMusic)
	os.Args = []string{"cmd", "-music", testMusic}
	ar = getArgs()
	assert.Equal(t, *ar.music, testMusic)

	// Test the usage of version flag
	os.Args = []string{"cmd", "-version"}
	ar = getArgs()
	assert.Equal(t, *ar.version, true)
}

func TestSetupHooks(t *testing.T) {

	gomu := newGomu()
	gomu.anko = anko.NewAnko()
	gomu.hook = hook.NewEventHook()

	err := loadModules(gomu.anko)
	if err != nil {
		t.Error(err)
	}

	setupHooks(gomu.hook, gomu.anko)

	const src = `
i = 0

Event.add_hook("skip", func() {
	i++
})
	`

	_, err = gomu.anko.Execute(src)
	if err != nil {
		t.Error(err)
	}

	gomu.hook.RunHooks("enter")

	for i := 0; i < 12; i++ {
		gomu.hook.RunHooks("skip")
	}

	got := gomu.anko.GetInt("i")

	assert.Equal(t, 12, got)
}

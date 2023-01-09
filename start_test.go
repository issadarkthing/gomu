package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/issadarkthing/gomu/anko"
	"github.com/issadarkthing/gomu/hook"
	"github.com/stretchr/testify/assert"
)

// Test default case
func TestGetArgsDefaults(t *testing.T) {
	args := getArgs()
	assert.Equal(t, *args.config, "~/.config/gomu/config")
	assert.Equal(t, *args.empty, false)
	assert.Equal(t, *args.music, "~/music")
	assert.Equal(t, *args.version, false)
}

// Test non-standard flags/the empty/version flags
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

	testMusic := filepath.Join(home, ".tmp", "gomu")
	_, err = os.Stat(testMusic)
	if os.IsNotExist(err) {
		os.MkdirAll(testMusic, 0755)
	}
	defer os.RemoveAll(testMusic)

	boolChecks := []struct {
		name string
		arg  bool
		want bool
	}{
		{"empty", true, true},
		{"version", true, true},
	}
	for _, check := range boolChecks {
		t.Run("testing bool flag "+check.name, func(t *testing.T) {
			flag.CommandLine.Set(check.name, strconv.FormatBool(check.arg))
			flag.CommandLine.Parse(os.Args[1:])
			assert.Equal(t, check.arg, check.want)
		})
	}
	strChecks := []struct {
		name string
		arg  string
		want string
	}{
		{"config", tmpCfgf.Name(), tmpCfgf.Name()},
		{"music", testMusic, testMusic},
	}
	for _, check := range strChecks {
		t.Run("testing string flag "+check.name, func(t *testing.T) {
			flag.CommandLine.Set(check.name, check.arg)
			flag.CommandLine.Parse(os.Args[1:])
			fmt.Println("flag value: ", check.arg)
			assert.Equal(t, check.arg, check.want)
		})
	}
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

package main

import (
	"testing"

	"github.com/issadarkthing/gomu/anko"
	"github.com/issadarkthing/gomu/hook"
	"github.com/stretchr/testify/assert"
)


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

package anko

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/anko/core"
	"github.com/mattn/anko/env"
	_ "github.com/mattn/anko/packages"
	"github.com/mattn/anko/vm"
)


type Anko struct {
	env *env.Env
}

func NewAnko() Anko {

	env := core.Import(env.NewEnv())
	importToX(env)

	return Anko{env}
}

// Define defines new symbol and value to the Anko env.
func (a *Anko) Define(symbol string, value interface{}) error {
	return a.env.DefineGlobal(symbol, value)
}

// Set sets new value to existing symbol. Use this when change value under an
// existing symbol.
func (a *Anko) Set(symbol string, value interface{}) error {
	return a.env.Set(symbol, value)
}

// Get gets value from anko env, returns error if symbol is not found.
func (a *Anko) Get(symbol string) (interface{}, error) {
	return a.env.Get(symbol)
}

// GetInt gets int value from symbol, returns golang default value if not found.
func (a *Anko) GetInt(symbol string) int {
	v, err := a.env.Get(symbol)
	if err != nil {
		return 0
	}

	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	}

	return 0
}

// GetString gets string value from symbol, returns golang default value if not
// found.
func (a *Anko) GetString(symbol string) string {
	v, err := a.env.Get(symbol)
	if err != nil {
		return ""
	}

	val, ok := v.(string)
	if !ok {
		return ""
	}

	return val
}

// GetBool gets bool value from symbol, returns golang default value if not
// found.
func (a *Anko) GetBool(symbol string) bool {
	v, err := a.env.Get(symbol)
	if err != nil {
		return false
	}

	val, ok := v.(bool)
	if !ok {
		return false
	}

	return val
}

// Execute executes anko script.
func (a *Anko) Execute(src string) (interface{}, error) {
	return vm.Execute(a.env, nil, src)
}

// KeybindExists checks if keybinding is defined.
func (a *Anko) KeybindExists(panel string, eventKey *tcell.EventKey) bool {
	var src string
	name := eventKey.Name()

	if strings.Contains(name, "Ctrl") {
		key := extractCtrlRune(name)
		src = fmt.Sprintf("Keybinds.%s.ctrl_%s", 
			panel, strings.ToLower(string(key)))

	} else if strings.Contains(name, "Alt") {
		key := extractAltRune(name)
		src = fmt.Sprintf("Keybinds.%s.alt_%c", panel, key)

	} else if strings.Contains(name, "Rune") {
		src = fmt.Sprintf("Keybinds.%s.%c", panel, eventKey.Rune())

	} else {
		src = fmt.Sprintf("Keybinds.%s.%s", panel, strings.ToLower(name))

	}

	val, err := a.Execute(src)
	if err != nil {
		return false
	}

	return val != nil
}

// ExecKeybind executes function bounded by the keybinding.
func (a *Anko) ExecKeybind(panel string, eventKey *tcell.EventKey) error {

	var src string
	name := eventKey.Name()

	if strings.Contains(name, "Ctrl") {
		key := extractCtrlRune(name)
		src = fmt.Sprintf("Keybinds.%s.ctrl_%s()", 
			panel, strings.ToLower(string(key)))

	} else if strings.Contains(name, "Alt") {
		key := extractAltRune(name)
		src = fmt.Sprintf("Keybinds.%s.alt_%c()", panel, key)

	} else if strings.Contains(name, "Rune") {
		src = fmt.Sprintf("Keybinds.%s.%c()", panel, eventKey.Rune())

	} else {
		src = fmt.Sprintf("Keybinds.%s.%s()", panel, strings.ToLower(name))

	}

	_, err := a.Execute(src)
	if err != nil {
		return err
	}

	return nil
}

func extractCtrlRune(str string) rune {
	re := regexp.MustCompile(`\+(.)$`)
	x := re.FindStringSubmatch(str)
	return rune(x[0][1])
}

func extractAltRune(str string) rune {
	re := regexp.MustCompile(`\[(.)\]`)
	x := re.FindStringSubmatch(str)
	return rune(x[0][1])
}

package anko

import (
	"fmt"

	"github.com/mattn/anko/core"
	"github.com/mattn/anko/env"
	"github.com/mattn/anko/vm"
	_ "github.com/mattn/anko/packages"
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
func (a *Anko) KeybindExists(panel string, keybind string) bool {
	src := fmt.Sprintf("Keybinds.%s.%s", panel, keybind)
	val, err := a.Execute(src)
	if err != nil {
		return false
	}

	return val != nil
}

// ExecKeybind executes function bounded by the keybinding.
func (a *Anko) ExecKeybind(panel string, keybind string) error {
	src := fmt.Sprintf("Keybinds.%s.%s()", panel, keybind)
	_, err := a.Execute(src)
	return err
}

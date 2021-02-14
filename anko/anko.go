package anko

import (
	"errors"
	"context"
	"reflect"
	"fmt"

	"github.com/mattn/anko/core"
	"github.com/mattn/anko/env"
	"github.com/mattn/anko/vm"
	_ "github.com/mattn/anko/packages"
)

var (
	ErrNoKeybind   = errors.New("no keybinding")
	ErrInvalidType = errors.New("invalid type")
)

type Anko struct {
	env *env.Env
}

func NewAnko() Anko {
	return Anko{
		core.Import(env.NewEnv()),
	}
}

// define defines new symbol and value to the Anko env
func (a *Anko) Define(symbol string, value interface{}) error {
	return a.env.DefineGlobal(symbol, value)
}

// set sets new value to existing symbol. Use this when change value under an
// existing symbol.
func (a *Anko) Set(symbol string, value interface{}) error {
	return a.env.Set(symbol, value)
}

// get gets value from anko env, returns error if symbol is not found.
func (a *Anko) Get(symbol string) (interface{}, error) {
	return a.env.Get(symbol)
}

// getInt gets int value from symbol, returns golang default value if not found
func (a *Anko) GetInt(symbol string) int {
	v, err := a.env.Get(symbol)
	if err != nil {
		return 0
	}

	val, ok := v.(int64)
	if !ok {
		return 0
	}

	return int(val)
}

// getString gets string value from symbol, returns golang default value if not
// found
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

// getBool gets bool value from symbol, returns golang default value if not
// found
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

// execute executes anko script
func (a *Anko) Execute(src string) (interface{}, error) {
	return vm.Execute(a.env, nil, src)
}

func (a *Anko) ExecKeybind(panel string, keybind string, cb func(error)) error {

	kb, err := a.Get("keybinds")
	if err != nil {
		return err
	}

	p, ok := kb.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("%w: require type {} got %T", ErrInvalidType, kb)
	}

	k, ok := p[panel]
	if !ok {
		return ErrNoKeybind
	}

	keybinds, ok := k.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("%w: require type {} got %T", ErrInvalidType, k)
	}

	cmd, ok := keybinds[keybind]
	if !ok {
		return ErrNoKeybind
	}

	f, ok := cmd.(func(context.Context) (reflect.Value, reflect.Value))
	if !ok {
		return fmt.Errorf("%w: require type func()", ErrInvalidType)
	}

	go func() {
		_, execErr := f(context.Background())
		if err := execErr.Interface(); !execErr.IsNil() {
			if err, ok := err.(error); ok {
				cb(err)
			}
		}
	}()


	return nil
}

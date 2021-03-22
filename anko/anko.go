package anko

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/anko/core"
	"github.com/mattn/anko/env"
	_ "github.com/mattn/anko/packages"
	"github.com/mattn/anko/parser"
	"github.com/mattn/anko/vm"
)

type Anko struct {
	env *env.Env
}

func NewAnko() *Anko {

	env := core.Import(env.NewEnv())
	importToX(env)

	t, err := env.Get("typeOf")
	if err != nil {
		panic(err)
	}

	k, err := env.Get("kindOf")
	if err != nil {
		panic(err)
	}

	env.DeleteGlobal("typeOf")
	env.DeleteGlobal("kindOf")

	err = env.Define("type_of", t)
	if err != nil {
		panic(err)
	}

	err = env.Define("kind_of", k)
	if err != nil {
		panic(err)
	}

	return &Anko{env}
}

// DefineGlobal defines new symbol and value to the Anko env.
func (a *Anko) DefineGlobal(symbol string, value interface{}) error {
	return a.env.DefineGlobal(symbol, value)
}

func (a *Anko) NewModule(name string) (*Anko, error) {
	env, err := a.env.NewModule(name)
	if err != nil {
		return nil, err
	}
	return &Anko{env}, nil
}

func (a *Anko) Define(name string, value interface{}) error {
	return a.env.Define(name, value)
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
	v, err := a.Execute(symbol)
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
	v, err := a.Execute(symbol)
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
	v, err := a.Execute(symbol)
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
	parser.EnableErrorVerbose()
	stmts, err := parser.ParseSrc(src)
	if err != nil {
		return nil, err
	}

	val, err := vm.Run(a.env, nil, stmts)
	if err != nil {
		if e, ok := err.(*vm.Error); ok {
			err = fmt.Errorf("error on line %d column %d: %s\n",
				e.Pos.Line, e.Pos.Column, err)
		} else if e, ok := err.(*parser.Error); ok {
			err = fmt.Errorf("error on line %d column %d: %s\n",
				e.Pos.Line, e.Pos.Column, err)
		}

		return nil, err
	}

	return val, nil
}

// KeybindExists checks if keybinding is defined.
func (a *Anko) KeybindExists(panel string, eventKey *tcell.EventKey) bool {
	var src string
	name := eventKey.Name()

	if strings.Contains(name, "Ctrl") {
		key := extractCtrlRune(name)
		src = fmt.Sprintf("Keybinds.%s[\"ctrl_%s\"]",
			panel, strings.ToLower(string(key)))

	} else if strings.Contains(name, "Alt") {
		key := extractAltRune(name)
		src = fmt.Sprintf("Keybinds.%s[\"alt_%c\"]", panel, key)

	} else if strings.Contains(name, "Rune") {
		src = fmt.Sprintf("Keybinds.%s[\"%c\"]", panel, eventKey.Rune())

	} else {
		src = fmt.Sprintf("Keybinds.%s[\"%s\"]", panel, strings.ToLower(name))

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
		src = fmt.Sprintf("Keybinds.%s[\"ctrl_%s\"]()",
			panel, strings.ToLower(string(key)))

	} else if strings.Contains(name, "Alt") {
		key := extractAltRune(name)
		src = fmt.Sprintf("Keybinds.%s[\"alt_%c\"]()", panel, key)

	} else if strings.Contains(name, "Rune") {
		src = fmt.Sprintf("Keybinds.%s[\"%c\"]()", panel, eventKey.Rune())

	} else {
		src = fmt.Sprintf("Keybinds.%s[\"%s\"]()", panel, strings.ToLower(name))

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

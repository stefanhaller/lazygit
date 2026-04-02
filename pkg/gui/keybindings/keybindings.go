package keybindings

import (
	"log"

	"github.com/gdamore/tcell/v3"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

func LabelFromKey(key gocui.Key) string {
	if !key.IsSet() {
		return ""
	}

	if key.KeyName() == gocui.KeyName(tcell.KeyRune) {
		return key.Str()
	}

	value, ok := config.LabelByKey[key.KeyName()]
	if ok {
		return value
	}

	return "unknown"
}

func GetKey(label string) gocui.Key {
	key, ok := config.KeyFromLabel(label)
	if !ok {
		log.Fatalf("Unrecognized key %s, this should have been caught by user config validation", label)
	}

	return key
}

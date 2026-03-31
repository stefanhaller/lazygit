package keybindings

import (
	"log"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/constants"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

func LabelFromKey(key gocui.Key) string {
	if !key.IsSet() {
		return ""
	}

	if key.KeyName() == gocui.KeyName(tcell.KeyRune) {
		return string(key.Ch())
	}

	value, ok := config.LabelByKey[key.KeyName()]
	if ok {
		return value
	}

	return "unknown"
}

func GetKey(key string) gocui.Key {
	if key == "<disabled>" {
		return gocui.Key{}
	}

	runeCount := utf8.RuneCountInString(key)
	if runeCount > 1 {
		keyName, ok := config.KeyByLabel[strings.ToLower(key)]
		if !ok {
			log.Fatalf("Unrecognized key %s for keybinding. For permitted values see %s", strings.ToLower(key), constants.Links.Docs.CustomKeybindings)
		}
		return gocui.NewKeyName(keyName)
	}

	if runeCount == 1 {
		return gocui.NewKeyRune([]rune(key)[0])
	}

	return gocui.Key{}
}

package stash

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var Stash = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Stashing files",
	ExtraCmdArgs: "",
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.CreateFile("file", "content")
		shell.GitAddAll()
	},
	Run: func(shell *Shell, input *Input, assert *Assert, keys config.KeybindingConfig) {
		assert.StashCount(0)

		input.PressKeys(keys.Files.ViewStashOptions)
		assert.InMenu()

		input.PressKeys("a")
		input.Type("stash name")
		input.Confirm()

		assert.StashCount(1)
	},
})

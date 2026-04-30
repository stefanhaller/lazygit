## Custom Keybindings

A keybinding is one of:

- A single printable character, e.g. `q`, `?`, `5`. Uppercase letters mean
  shift+letter — write `A`, not `<s-a>`.
- A special key name in angle brackets, e.g. `<enter>`, `<f1>`, `<up>`.
- A key with modifiers in angle brackets, e.g. `<c-c>` (Ctrl+C), `<c-s-up>`
  (Ctrl+Shift+Up).
- The literal string `<disabled>` to disable a binding.

### Modifiers

Prefix a key with one or more modifiers, joined by `-`:

| Prefix | Long form | Modifier                                                                                  |
| ------ | --------- | ----------------------------------------------------------------------------------------- |
| `c-`   | `ctrl-`   | Ctrl                                                                                      |
| `a-`   | `alt-`    | Alt                                                                                       |
| `s-`   | `shift-`  | Shift                                                                                     |
| `m-`   | `meta-`   | Depends on terminal; typically ⌘ on macOS or Super/Win key, when the terminal forwards it |

You can also use `+` instead of `-` as the separator. Modifiers may appear in
any order, and short and long forms can be mixed. The whole binding should be
wrapped in angle brackets when it has any modifiers. The following all express
the same binding:

- `<c-s-up>`
- `<ctrl-shift-up>`
- `<ctrl+shift+up>`
- `<s-c-up>`

### Special key names

| Put in                                  | You will get        |
| --------------------------------------- | ------------------- |
| `<f1>` – `<f12>`                        | F1 – F12            |
| `<insert>`                              | Insert              |
| `<delete>`                              | Delete              |
| `<home>`                                | Home                |
| `<end>`                                 | End                 |
| `<pgup>`                                | PageUp              |
| `<pgdown>`                              | PageDown            |
| `<up>`                                  | ArrowUp             |
| `<down>`                                | ArrowDown           |
| `<left>`                                | ArrowLeft           |
| `<right>`                               | ArrowRight          |
| `<tab>`                                 | Tab                 |
| `<backtab>`                             | Shift+Tab           |
| `<enter>`                               | Enter               |
| `<esc>`                                 | Escape              |
| `<backspace>`                           | Backspace           |
| `<space>`                               | Space               |
| `<mouse wheel up>`/`<mouse wheel down>` | Mouse wheel up/down |

These can be combined with modifiers, e.g. `<c-up>`, `<c-s-f1>`, `<a-enter>`.

### Special characters with modifiers

`<minus>` and `<plus>` are keyword forms for `-` and `+` when combined with a
modifier (e.g. `<c-minus>` for Ctrl+`-`). Without modifiers, write `-` and `+`
directly. `<space>` is the keyword for the space character.

### Combinations that are rejected

These look reasonable but can't actually be delivered by a terminal:

- `<s-a>` (shift alone on a rune) — terminals fold shift into the rune itself,
  so shift+a arrives as `A`. Write `A` instead.
- `<c-A>`, `<a-A>`, etc. (modifier on an uppercase ASCII letter) — write
  `<c-s-a>` instead.

### Terminal compatibility

Support for combinations of modifiers, and in general keybindings beyond plain
letters and ctrl+letter, require a newer terminal protocol that not all
terminals support.

Terminals that are known to have good support include: Ghostty, kitty,
WezTerm, foot, Konsole, Alacritty, iTerm2, Windows Terminal.

The default terminal on macOS (Terminal.app) does not; I recommend to switch to
either Ghostty or iTerm2 as a replacement (or one of the others above).

On Windows, a popular terminal is the MinTTY console that comes with Git for
Windows; this also doesn't support the newer protocol. The recommended
replacement is Windows Terminal, which is very good these days, and Git Bash
runs just fine in it.

Inside **tmux** or **screen**, extended keys are stripped unless the multiplexer
is configured to forward them. For tmux 3.2+:

​`
set -g extended-keys on
set -as terminal-features 'xterm*:extkeys'
​`

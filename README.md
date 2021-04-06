# tmux-wormhole

Use tmux and magic wormhole to get things from your remote computer to your tmux. If tmux 
has DISPLAY set, open the file locally!

## Demo

![tmux-wormhole](https://user-images.githubusercontent.com/45680/113491108-37fbf400-949c-11eb-80f5-829b045f1701.gif)

## Usage

- On your remote computer, display the magic wormhole code.
- Press ( <kbd>prefix</kbd> + <kbd>w</kbd> )
- Hit OK to transfer.

## Prerequisites

`tmux-wormhole` is written in Go. To install `tmux-wormhole` successfully, you'll need Go version 1.13 or higher.

## Setup with Tmux Plugin Manager

Set up this plugin via [TPM](https://github.com/tmux-plugins/tpm) by adding this to your `~/.tmux.conf`:

```
set -g @plugin 'gcla/tmux-wormhole'
```

Install the plugin by hitting <kbd>prefix</kbd> + <kbd>I</kbd>. 

## Setup Manually

Clone the repo:

```
git clone https://github.com/gcla/tmux-wormhole ~/.tmux/plugins/tmux-wormhole
```

Compile it:

```
cd ~/.tmux/plugins/tmux-wormhole
GO11MODULE=on go build -o tmux-wormhole cmd/tmux-wormhole/main.go
```

Source it by adding this to your `~/.tmux.conf`:

```
run-shell ~/.tmux/plugins/tmux-wormhole/tmux-wormhole.tmux
```

Reload TMUX's config with:

```
tmux source-file ~/.tmux.conf
```

## Configuration

Set these in your `~/.tmux.conf` file.

- @wormhole-key - how to launch tmux-wormhole (default: `w`)
- @wormhole-save-folder - where to keep transferred files and directories (default: XDG download dir e.g. `~/Downloads/`)
- @wormhole-open-cmd - run this command after a file is transferred (default: `xdg-open` or `open`)
- @wormhole-no-default-open - just transfer, don't run anything afterwards (default: `false`)
- @wormhole-no-ask-to-open - after a file is transferred, ask the user interactively if the file should be opened (default: `false`)
- @wormhole-can-overwrite - allow tmux-wormhole to overwite a file or directory of the same name locally (default: `false`)

## How does it work

The plugin uses sleight of hand to make it look as though its prompts are being displayed over the active pane. When you hit the tmux-wormhole hotkey,
the plugin does the following:

- saves the contents of the active pane to a temporary file e.g. `/tmp/wormhole`
- launches a new tmux session called `wormhole`, with...
- a pane running `cat /tmp/wormhole ; sleep infinity`

If you were to attach to the wormhole session, this pane should look like the currently active pane. Next the plugin will:

- create a new window called `wormhole-ABC` in the active session 
- with a single pane running the Go program `tmux-wormhole`
- `tmux-wormhole` is a gowid application that overlays a dialog widget on top of a terminal widget. The terminal widget runs `tmux attach -L wormhole`

Finally, the plugin swaps the currently active pane with the pane from `wormhole-ABC`.

The effect is that the terminal now has a yellow dialog overlaid.

## Sources

- [tmux-thumbs](https://github.com/fcsonline/tmux-thumbs) for the project structure which I freely plagiarized!
- [wormhole-william](https://github.com/psanford/wormhole-william) - a GoLang implementation of magic wormhole
- [gowid](https://github.com/gcla/gowid) - my Go TUI framework, which itself heavily depends on...
- [tcell](https://github.com/gdamore/tcell) - like ncurses for GoLang

# License

[MIT](https://github.com/fcsonline/tmux-thumbs/blob/master/LICENSE)

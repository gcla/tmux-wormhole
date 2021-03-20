// Copyright 2021 Graham Clark. All rights reserved.  Use of this source
// code is governed by the MIT license that can be found in the LICENSE
// file.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	_ "net/http"
	_ "net/http/pprof"

	"github.com/adrg/xdg"
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/holder"
	"github.com/gcla/gowid/widgets/selectable"
	"github.com/gcla/gowid/widgets/terminal"
	"github.com/gcla/tmux-wormhole/pkg/widgets/hilite"
	"github.com/gcla/tmux-wormhole/pkg/wormflow"
	"github.com/gdamore/tcell"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

//======================================================================

var app *gowid.App
var term *terminal.Widget
var code string
var saveDir string
var session string
var shell string
var openCmd string
var willQuit bool

//======================================================================

// Go's main() prototype does not provide for returning a value.
func main() {
	res := cmain()
	os.Exit(res)
}

//======================================================================

func envTrue(val string) bool {
	switch strings.ToLower(val) {
	case "true", "t", "yes", "y", "1":
		return true
	default:
		return false
	}
}

func quit(app gowid.IApp) {
	if !willQuit {
		willQuit = true
		app.Quit()
	}
}

type handler struct {
	controller *wormflow.Controller
}

func (h handler) UnhandledInput(app gowid.IApp, ev interface{}) bool {
	handled := false

	if evk, ok := ev.(*tcell.EventKey); ok {
		switch evk.Key() {
		case tcell.KeyCtrlC, tcell.KeyEsc:
			handled = true
			quit(app)
		}
	}
	return handled
}

//======================================================================

func cmain() int {
	var err error

	// Do these before we switch the terminal to graphics
	shell = os.Getenv("SHELL")
	if shell == "" {
		fmt.Printf("This tmux plugin requires a value in the env variable SHELL.\n")
		return 1
	}

	code = os.Getenv("TMUX_WORMHOLE_CODE")
	// If code is empty, it means the bash wrapper didn't find one. Show that error in the UI
	// which means we need to launch the UI first. But I do assume that what is provided is
	// either empty or will compile to a regexp. If not, it's ok to show a low-tech error.
	codeRe, err := regexp.Compile(code)
	if err != nil {
		fmt.Printf("Wormhole code %s is invalid.\n", code)
		return 1
	}

	session = os.Getenv("TMUX_WORMHOLE_SESSION")
	if session == "" {
		fmt.Printf("This tmux plugin requires a value in the env variable TMUX_WORMHOLE_SESSION.\n")
		return 1
	}

	saveDir = os.Getenv("TMUX_WORMHOLE_SAVE_FOLDER")
	if saveDir == "" {
		saveDir = xdg.UserDirs.Download
	}
	if saveDir == "" {
		saveDir = "."
	}
	if saveDir != "" {
		saveDir, err = homedir.Expand(saveDir)
		if err != nil {
			fmt.Printf("Problem expanding save directory %s: %v\n", saveDir, err)
			return 1
		}
	}

	// Takes precedence
	openCmd = os.Getenv("TMUX_WORMHOLE_OPEN_CMD")
	if openCmd == "" && !envTrue(os.Getenv("TMUX_WORMHOLE_NO_DEFAULT_OPEN")) {
		switch runtime.GOOS {
		case "darwin":
			openCmd = "open"
		case "linux", "dragonfly", "freebsd", "netbsd", "openbsd":
			openCmd = "xdg-open"
		}
	}

	// Avoid gowid's dim screen problem with truecolor - need to fix
	os.Setenv("COLORTERM", "")

	palette := gowid.Palette{
		"dialog":            gowid.MakeStyledPaletteEntry(gowid.ColorBlack, gowid.ColorYellow, gowid.StyleNone),
		"dialog-button":     gowid.MakeStyledPaletteEntry(gowid.ColorYellow, gowid.ColorBlack, gowid.StyleNone),
		"button":            gowid.MakeForeground(gowid.ColorMagenta),
		"button-focus":      gowid.MakePaletteEntry(gowid.ColorWhite, gowid.ColorDarkBlue),
		"progress-default":  gowid.MakeStyledPaletteEntry(gowid.ColorWhite, gowid.ColorBlack, gowid.StyleBold),
		"progress-complete": gowid.MakeStyleMod(gowid.MakePaletteRef("progress-default"), gowid.MakeBackground(gowid.ColorMagenta)),
		"progress-spinner":  gowid.MakePaletteEntry(gowid.ColorMagenta, gowid.ColorBlack),
	}

	hkDuration := terminal.HotKeyDuration{time.Second * 3}

	term, err = terminal.NewExt(terminal.Options{
		Command: []string{
			shell, "-c", fmt.Sprintf("tmux -L wormhole attach-session -t '%s'", session),
		},
		HotKeyPersistence: &hkDuration,
		Scrollback:        100,
	})
	if err != nil {
		fmt.Printf("Unexpected error running gowid terminal widget: %v\n", err)
		return 1
	}

	term.OnProcessExited(gowid.WidgetCallback{"cb",
		func(app gowid.IApp, w gowid.IWidget) {
			quit(app)
		},
	})

	// Don't want any user input going to the pane below the dialog, which is really
	// a mock-up of the pane that was being displayed before the plugin ran.
	h := holder.New(
		selectable.NewUnselectable(
			hilite.New(term, codeRe, hilite.Options{
				Background: gowid.ColorGreen,
				Foreground: gowid.ColorBlack,
			}),
		),
	)

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	app, err = gowid.NewApp(gowid.AppArgs{
		View:    h,
		Palette: &palette,
		Log:     log,
	})

	if err != nil {
		fmt.Printf("Unexpected error launching gowid app: %v\n", err)
		return 1
	}

	controller := wormflow.New(wormflow.Args{
		Code:      code,
		SaveDir:   saveDir,
		OpenCmd:   openCmd,
		NoAskOpen: envTrue(os.Getenv("TMUX_WORMHOLE_NO_ASK_TO_OPEN")),
		Overwrite: envTrue(os.Getenv("TMUX_WORMHOLE_CAN_OVERWRITE")),
		Shell:     shell,
		Lower:     h,
	})

	controller.Start(app)

	app.MainLoop(handler{controller: controller})

	return 0
}

//======================================================================
// Local Variables:
// mode: Go
// fill-column: 78
// End:

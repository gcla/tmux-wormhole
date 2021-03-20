// Copyright 2021 Graham Clark. All rights reserved.  Use of this source
// code is governed by the MIT license that can be found in the LICENSE
// file.

// Package wormflow contains code that provides the UI for tmux-wormhole's
// magic-wormhole file-receiving feature.
package wormflow

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/gwutil"
	"github.com/gcla/gowid/widgets/dialog"
	"github.com/gcla/gowid/widgets/divider"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/hpadding"
	"github.com/gcla/gowid/widgets/pile"
	"github.com/gcla/gowid/widgets/progress"
	"github.com/gcla/gowid/widgets/spinner"
	"github.com/gcla/gowid/widgets/text"
	"github.com/psanford/wormhole-william/wormhole"
)

//======================================================================

type Args struct {
	Code      string
	SaveDir   string
	OpenCmd   string
	NoAskOpen bool
	Shell     string
	Overwrite bool
	Lower     gowid.ISettableComposite
}

type Controller struct {
	Args
}

// Transfer exists to provide a simple description of the wormhole transfer type
// for displaying to the user.
type Transfer wormhole.TransferType

func (tt Transfer) String() string {
	switch wormhole.TransferType(tt) {
	case wormhole.TransferFile:
		return "file"
	case wormhole.TransferDirectory:
		return "directory"
	case wormhole.TransferText:
		return "message"
	default:
		return "unknown message"
	}
}

//======================================================================

func New(args Args) *Controller {
	res := &Controller{
		Args: args,
	}
	return res
}

func (w *Controller) Start(app gowid.IApp) {
	app.Run(gowid.RunFunction(func(app gowid.IApp) {
		if w.Args.Code == "" {
			w.noCode(app)
		} else {
			w.displayCode(app)
		}
	}))
}

//======================================================================

func makeTxtDialog(txt string, buttons ...dialog.Button) *dialog.Widget {
	return makeDialog(text.New(txt), gowid.RenderFixed{}, buttons...)
}

func makeDialog(w gowid.IWidget, wid gowid.IWidgetDimension, buttons ...dialog.Button) *dialog.Widget {
	d := &dialog.Widget{}

	for _, b := range buttons {
		b.Action.(iPrevious).SetPrevious(d)
	}

	*d = *dialog.New(
		framed.NewSpace(
			hpadding.New(
				w,
				gowid.HAlignMiddle{},
				wid,
			)),
		dialog.Options{
			Buttons:         buttons,
			NoEscapeClose:   true,
			NoShadow:        true,
			BackgroundStyle: gowid.MakePaletteRef("dialog"),
			BorderStyle:     gowid.MakePaletteRef("dialog"),
			ButtonStyle:     gowid.MakePaletteRef("dialog-button"),
		},
	)
	return d
}

//======================================================================

type iPrevious interface {
	SetPrevious(*dialog.Widget)
}

type common struct {
	previous *dialog.Widget // so it can be closed
}

func (d common) ID() interface{} {
	return "dummy"
}

func (d *common) SetPrevious(p *dialog.Widget) {
	d.previous = p
}

//======================================================================

type quit struct {
	common
	*Controller
}

// Show the code - hit Cancel button
func (w quit) Changed(app gowid.IApp, widget gowid.IWidget, data ...interface{}) {
	app.Quit()
}

//======================================================================

type showCodeOk struct {
	common
	*Controller
}

// Show the code - hit Ok button
func (w showCodeOk) Changed(app gowid.IApp, widget gowid.IWidget, data ...interface{}) {
	var client wormhole.Client

	// goroutine so I don't block ui goroutine
	go func() {
		ctx := context.Background()
		msg, err := client.Receive(ctx, w.Args.Code)
		reject := true

		if err != nil {
			app.Run(gowid.RunFunction(func(app gowid.IApp) {
				w.previous.Close(app)
				w.doReceiveError(err, app)
			}))
			return
		}

		defer func() {
			//fmt.Fprintf(os.Stderr, "GCLA: will I reject? reject is %v\n", reject)
			if reject {
				msg.Reject()
			}
		}()

		switch msg.Type {
		case wormhole.TransferText:

			// Wormhole william doesn't allow rejecting text message
			// transfers
			reject = false

			spin := spinner.New(spinner.Options{
				Styler: gowid.MakePaletteRef("progress-spinner"),
			})

			app.Run(gowid.RunFunction(func(app gowid.IApp) {
				w.previous.Close(app)
				w.doSpin(spin, app)
			}))

			done := make(chan struct{})
			var transferredMessage []byte

			go func() {
				transferredMessage, err = ioutil.ReadAll(msg)

				defer close(done)

				if err != nil {
					app.Run(gowid.RunFunction(func(app gowid.IApp) {
						w.previous.Close(app)
						w.doTextTransferError(err, app)
					}))
					return
				}

				// Artificial delay makes a nicer experience
				time.Sleep(1000 * time.Millisecond)
			}()

			go func() {
				c := time.Tick(100 * time.Millisecond)
			loop:
				for {
					select {
					case <-done:
						app.Run(gowid.RunFunction(func(app gowid.IApp) {
							w.previous.Close(app)
							w.doMessageThenQuit(string(transferredMessage), "Quit", app)
						}))
						break loop
					case <-c:
						app.Run(gowid.RunFunction(func(app gowid.IApp) {
							spin.Update()
						}))
					}
				}
			}()

		case wormhole.TransferFile:
			prog := progress.New(progress.Options{
				Normal:   gowid.MakePaletteRef("progress-default"),
				Complete: gowid.MakePaletteRef("progress-complete"),
			})

			app.Run(gowid.RunFunction(func(app gowid.IApp) {
				w.previous.Close(app)
				w.doProg(msg.Name, Transfer(msg.Type), prog, app)
				//w.InTransfer = true
			}))

			done := make(chan struct{})
			read := 0

			// Only set if dir or file
			savedFilename := filepath.Join(w.Args.SaveDir, msg.Name)
			if !w.Args.Overwrite && fileExists(savedFilename) {
				app.Run(gowid.RunFunction(func(app gowid.IApp) {
					w.previous.Close(app)
					w.doNoOverwrite(savedFilename, app)
				}))
				return
			}

			f, err := os.Create(savedFilename)
			if err != nil {
				app.Run(gowid.RunFunction(func(app gowid.IApp) {
					w.previous.Close(app)
					w.doFileCreateError(savedFilename, err, app)
				}))
				return
			}

			reject = false

			go func() {
				_, err = io.Copy(f, &progReader{read: &read, Reader: msg})

				defer func() {
					f.Close()
					close(done)
				}()

				if err != nil {
					app.Run(gowid.RunFunction(func(app gowid.IApp) {
						w.previous.Close(app)
						w.doFileTransferError(savedFilename, err, app)
					}))
					return
				}
			}()

			go func() {
				c := time.Tick(250 * time.Millisecond)
			loop:
				for {
					select {
					case <-done:
						app.Run(gowid.RunFunction(func(app gowid.IApp) {
							prog.SetTarget(app, int(msg.UncompressedBytes64))
							prog.SetProgress(app, int(msg.UncompressedBytes64))

							// Delay at 100% is nice
							time.AfterFunc(1*time.Second, func() {
								app.Run(gowid.RunFunction(func(app gowid.IApp) {
									//w.InTransfer = false
									w.previous.Close(app)

									if w.Args.OpenCmd == "" {
										w.doSavedAs(savedFilename, app)
									} else {
										if w.Args.NoAskOpen {
											w.doOpen(savedFilename, app)
										} else {
											w.doAskToOpen(savedFilename, app)
										}
									}
								}))
							})

						}))
						break loop
					case <-c:
						app.Run(gowid.RunFunction(func(app gowid.IApp) {
							prog.SetTarget(app, int(msg.UncompressedBytes64))
							prog.SetProgress(app, read)
						}))
					}
				}
			}()

		case wormhole.TransferDirectory:

			dirName := filepath.Join(w.Args.SaveDir, msg.Name)

			err = os.Mkdir(dirName, 0777)
			if err != nil {
				app.Run(gowid.RunFunction(func(app gowid.IApp) {
					w.previous.Close(app)
					w.doError(err, app)
				}))
				return
			}

			tmpFile, err := ioutil.TempFile(w.Args.SaveDir, fmt.Sprintf("%s.zip.tmp", msg.Name))
			if err != nil {
				app.Run(gowid.RunFunction(func(app gowid.IApp) {
					w.previous.Close(app)
					w.doError(err, app)
				}))
				return
			}

			prog := progress.New(progress.Options{
				Normal:   gowid.MakePaletteRef("progress-default"),
				Complete: gowid.MakePaletteRef("progress-complete"),
			})

			app.Run(gowid.RunFunction(func(app gowid.IApp) {
				w.previous.Close(app)
				w.doProg(msg.Name, Transfer(msg.Type), prog, app)
			}))

			done := make(chan struct{})
			read := 0

			reject = false

			go func() {

				errme := func(w showCodeOk, err error, app gowid.IApp) {
					app.Run(gowid.RunFunction(func(app gowid.IApp) {
						w.previous.Close(app)
						w.doError(err, app)
					}))
				}

				defer func() {
					tmpFile.Close()
					os.Remove(tmpFile.Name())
					close(done)
				}()

				n, err := io.Copy(tmpFile, &progReader{read: &read, Reader: msg})

				if err != nil {
					app.Run(gowid.RunFunction(func(app gowid.IApp) {
						w.previous.Close(app)
						w.doFileTransferError(msg.Name, err, app)
					}))
					return
				}

				tmpFile.Seek(0, io.SeekStart)
				zr, err := zip.NewReader(tmpFile, int64(n))
				if err != nil {
					app.Run(gowid.RunFunction(func(app gowid.IApp) {
						w.previous.Close(app)
						w.doError(err, app)
					}))
					return
				}

				for _, zf := range zr.File {
					p, err := filepath.Abs(filepath.Join(dirName, zf.Name))
					if err != nil {
						errme(w, err, app)
						return
					}

					if !strings.HasPrefix(p, dirName) {
						app.Run(gowid.RunFunction(func(app gowid.IApp) {
							w.previous.Close(app)
							w.doMessageThenQuit(fmt.Sprintf("Dangerous filename found: %s", zf.Name), "Quit", app)
						}))
						return
					}

					rc, err := zf.Open()
					if err != nil {
						app.Run(gowid.RunFunction(func(app gowid.IApp) {
							w.previous.Close(app)
							w.doMessageThenQuit(fmt.Sprintf("%s open failed: %v", zf.Name, err), "Quit", app)
						}))
						return
					}

					dir := filepath.Dir(p)
					err = os.MkdirAll(dir, 0777)
					if err != nil {
						errme(w, err, app)
						return
					}

					f, err := os.Create(p)
					if err != nil {
						errme(w, err, app)
						return
					}

					_, err = io.Copy(f, rc)
					if err != nil {
						errme(w, err, app)
						return
					}

					err = f.Close()
					if err != nil {
						errme(w, err, app)
						return
					}

					rc.Close()
				}
			}()

			go func() {
				c := time.Tick(250 * time.Millisecond)
			loop:
				for {
					select {
					case <-done:
						app.Run(gowid.RunFunction(func(app gowid.IApp) {
							prog.SetTarget(app, int(msg.UncompressedBytes64))
							prog.SetProgress(app, int(msg.UncompressedBytes64))

							// Delay at 100% is nice
							time.AfterFunc(1*time.Second, func() {
								app.Run(gowid.RunFunction(func(app gowid.IApp) {
									w.previous.Close(app)
									w.doSavedAs(dirName, app)
								}))
							})

						}))
						break loop
					case <-c:
						app.Run(gowid.RunFunction(func(app gowid.IApp) {
							prog.SetTarget(app, int(msg.UncompressedBytes64))
							prog.SetProgress(app, read)
						}))
					}
				}
			}()

		}

	}()
}

//======================================================================

func (w *Controller) openSaveError(savedFilename string, cmd string, err error, app gowid.IApp) {
	txt := fmt.Sprintf("Error opening: %s: %v", cmd, err)
	d := makeTxtDialog(txt,
		dialog.Button{
			Msg:    "Continue",
			Action: &savedAs{savedFilename: savedFilename, Controller: w},
		},
	)

	dialog.OpenExt(d, w.Lower, gowid.RenderWithUnits{U: len(txt) + 10}, gowid.RenderFlow{}, app)
}

//======================================================================

type savedAs struct {
	common
	savedFilename string
	*Controller
}

// Show the code - hit Cancel button
func (w savedAs) Changed(app gowid.IApp, widget gowid.IWidget, data ...interface{}) {
	w.previous.Close(app)
	w.doSavedAs(w.savedFilename, app)
}

func (w *Controller) doSavedAs(savedFilename string, app gowid.IApp) {
	w.doMessageThenQuit(fmt.Sprintf("Saved as %s", savedFilename), "Ok", app)
}

//======================================================================

func (w *Controller) doProg(name string, trans Transfer, prog *progress.Widget, app gowid.IApp) {
	txt := fmt.Sprintf("Transferring %s %s...", trans, name)

	rows := pile.NewFlow(
		text.New(txt),
		divider.NewBlank(),
		prog,
	)

	d := makeDialog(rows,
		gowid.RenderFlow{},
		dialog.Button{
			Msg: "Cancel",
			// Can't really cancel, can't interrupt receive
			Action: &quit{Controller: w},
		},
	)

	dialog.OpenExt(d, w.Lower, gowid.RenderWithUnits{U: gwutil.Max(32, len(txt)+10)}, gowid.RenderFlow{}, app)
}

//======================================================================

func (w *Controller) doSpin(spin *spinner.Widget, app gowid.IApp) {
	txt := "Transferring message..."

	rows := pile.NewFlow(
		text.New(txt),
		divider.NewBlank(),
		spin,
	)

	d := makeDialog(rows,
		gowid.RenderFlow{},
		dialog.Button{
			Msg:    "Cancel",
			Action: &quit{Controller: w},
		},
	)

	dialog.OpenExt(d, w.Lower, gowid.RenderWithUnits{U: gwutil.Min(32, len(txt)+10)}, gowid.RenderFlow{}, app)
}

//======================================================================

func (w *Controller) doReceiveError(err error, app gowid.IApp) {
	w.doMessageThenQuit(fmt.Sprintf("Error: %v", err), "Quit", app)
}

//======================================================================

func (w *Controller) doFileCreateError(filename string, err error, app gowid.IApp) {
	w.doMessageThenQuit(fmt.Sprintf("Error creating %s: %v", filename, err), "Quit", app)
}

//======================================================================

func (w *Controller) doFileTransferError(filename string, err error, app gowid.IApp) {
	w.doMessageThenQuit(fmt.Sprintf("Error transferring %s: %v", filename, err), "Quit", app)
}

//======================================================================

func (w *Controller) doTextTransferError(err error, app gowid.IApp) {
	w.doMessageThenQuit(fmt.Sprintf("Error transferring message: %v", err), "Quit", app)
}

//======================================================================

func (w *Controller) noCode(app gowid.IApp) {
	w.doMessageThenQuit("No wormhole code found!", "Quit", app)
}

//======================================================================

func (w *Controller) doNoOverwrite(savedFilename string, app gowid.IApp) {
	w.doMessageThenQuit(fmt.Sprintf("%s exists. Will not overwrite.", savedFilename), "Quit", app)
}

//======================================================================

func (w *Controller) doError(err error, app gowid.IApp) {
	w.doMessageThenQuit(fmt.Sprintf("Error: %v", err), "Quit", app)
}

//======================================================================

func (w *Controller) doMessageThenQuit(message string, label string, app gowid.IApp) {
	txt := fmt.Sprintf("%s", message)
	d := makeTxtDialog(txt,
		dialog.Button{
			Msg:    label,
			Action: &quit{Controller: w},
		},
	)

	dialog.OpenExt(d, w.Lower, gowid.RenderWithUnits{U: len(txt) + 10}, gowid.RenderFlow{}, app)
}

//======================================================================

func (w *Controller) displayCode(app gowid.IApp) {
	txt := fmt.Sprintf("%s. Proceed?", w.Args.Code)

	d := makeTxtDialog(txt,
		dialog.Button{
			Msg:    "Ok",
			Action: &showCodeOk{Controller: w},
		},
		dialog.Button{
			Msg:    "Cancel",
			Action: &quit{Controller: w},
		},
	)

	dialog.OpenExt(d, w.Lower, gowid.RenderWithUnits{U: len(txt) + 10}, gowid.RenderFlow{}, app)
}

//======================================================================

type open struct {
	common
	savedFilename string
	*Controller
}

func (w open) Changed(app gowid.IApp, widget gowid.IWidget, data ...interface{}) {
	w.previous.Close(app)
	w.doOpen(w.savedFilename, app)
}

func (w *Controller) doOpen(savedFilename string, app gowid.IApp) {
	var shellCmd string
	if strings.Contains(w.Args.OpenCmd, "%s") {
		shellCmd = strings.Replace(w.Args.OpenCmd, "%s", shellescape.Quote(savedFilename), -1)
	} else {
		shellCmd = w.Args.OpenCmd + " " + shellescape.Quote(savedFilename)
	}
	err := exec.Command(w.Args.Shell, "-c", shellCmd).Run()

	if err == nil {
		w.doSavedAs(savedFilename, app)
	} else {
		w.openSaveError(savedFilename, shellCmd, err, app)
	}
}

//======================================================================

func (w *Controller) doAskToOpen(savedFilename string, app gowid.IApp) {
	txt := fmt.Sprintf("Open %s?", savedFilename)
	d := makeTxtDialog(txt,
		dialog.Button{
			Msg:    "Yes",
			Action: &open{savedFilename: savedFilename, Controller: w},
		},
		dialog.Button{
			Msg:    "No",
			Action: &savedAs{savedFilename: savedFilename, Controller: w},
		},
	)

	dialog.OpenExt(d, w.Lower, gowid.RenderWithUnits{U: len(txt) + 10}, gowid.RenderFlow{}, app)
}

//======================================================================

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

//======================================================================

type progReader struct {
	read *int
	io.Reader
}

func (r *progReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	(*r.read) += n
	return
}

//======================================================================
// Local Variables:
// mode: Go
// fill-column: 110
// End:

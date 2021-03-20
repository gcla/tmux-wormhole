// Copyright 2021 Graham Clark. All rights reserved.  Use of this source
// code is governed by the MIT license that can be found in the LICENSE
// file.

// Package hilite contains a widget that hilights matching sections of text
// by scanning the rendered canvas.
package hilite

import (
	"regexp"

	"github.com/gcla/gowid"
)

//======================================================================

type Options struct {
	Background gowid.TCellColor
	Foreground gowid.TCellColor
}

type Widget struct {
	gowid.IWidget
	Match *regexp.Regexp
	Opt   Options
}

var _ gowid.IWidget = (*Widget)(nil)

//======================================================================

func New(inner gowid.IWidget, re *regexp.Regexp, opts ...Options) *Widget {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	} else {
		opt = Options{
			Background: gowid.ColorLightGreen,
			Foreground: gowid.ColorBlack,
		}
	}

	res := &Widget{
		IWidget: inner,
		Match:   re,
		Opt:     opt,
	}
	return res
}

func (w *Widget) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	res := w.IWidget.Render(size, focus, app)

	cbytes := canvasToArray(res)

	matches := w.Match.FindAllIndex(cbytes, -1)

	wid := res.BoxColumns()
	if wid < 1 {
		return res
	}

	var x int
	var y int
	for _, m := range matches {
		for j := m[0]; j < m[1]; j++ {
			x = j % wid
			y = j / wid
			res.SetCellAt(x, y, res.CellAt(x, y).WithBackgroundColor(w.Opt.Background).WithForegroundColor(w.Opt.Foreground))
		}
	}

	return res
}

func canvasToArray(c gowid.ICanvas) []byte {
	res := make([]byte, c.BoxRows()*c.BoxColumns())
	var r rune
	n := 0
	for i := 0; i < c.BoxRows(); i++ {
		for j := 0; j < c.BoxColumns(); j++ {
			r = c.CellAt(j, i).Rune()
			if int(r) < 32 || int(r) > 127 {
				r = ' '
			}
			res[n] = byte(r)
			n++
		}
	}
	return res
}

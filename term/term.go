// Copyright 2022 Linka Cloud  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package term

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
	"unicode/utf8"

	"go.linka.cloud/console"
)

var _ Term = (*terminal)(nil)

var (
	ExitRune = '\x1D'
)

type Size struct {
	Rows int
	Cols int
}

type Term interface {
	io.ReadWriteCloser
	Size() Size
	WatchSize() <-chan Size
}

type terminal struct {
	in      io.Reader
	console console.Console

	size  Size
	mu    sync.RWMutex
	sch   chan Size
	sonce sync.Once

	close chan struct{}
	conce sync.Once
}

func New(ctx context.Context) (Term, error) {
	c := console.Current()
	if err := c.SetRaw(); err != nil {
		return nil, err
	}
	ws, err := c.Size()
	if err != nil {
		return nil, err
	}
	if err := c.Resize(ws); err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	r := io.TeeReader(c, pw)
	term := &terminal{
		in:      r,
		console: c,
		size:    Size{Rows: int(ws.Height), Cols: int(ws.Width)},
		close:   make(chan struct{}),
	}

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			if err := ctx.Err(); err != nil {
				return
			}
			nws, err := c.Size()
			if err != nil {
				continue
			}
			if nws.Height == ws.Height && nws.Width == ws.Width {
				continue
			}
			ws = nws
			term.mu.Lock()
			term.size = Size{Rows: int(ws.Height), Cols: int(ws.Width)}
			term.mu.Unlock()

			term.mu.RLock()
			if term.sch != nil {
				term.sch <- term.size
			}
			term.mu.RUnlock()
		}
	}()

	go func() {
		defer term.Close()
		buf := make([]byte, 512)
		for {
			n, err := pr.Read(buf)
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				return
			}
			if n == 0 {
				continue
			}
			if r, _ := utf8.DecodeRune(buf[:n]); r == ExitRune {
				return
			}
		}
	}()

	return term, nil
}

func (s *terminal) Read(p []byte) (n int, err error) {
	return s.in.Read(p)
}

func (s *terminal) Write(p []byte) (n int, err error) {
	return s.console.Write(p)
}

func (s *terminal) Size() Size {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

func (s *terminal) WatchSize() <-chan Size {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sch == nil {
		s.sch = make(chan Size, 1)
	}
	return s.sch
}

func (s *terminal) Close() error {
	var err error
	s.conce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		err = s.console.Reset()
		if s.sch != nil {
			close(s.sch)
		}
		close(s.close)
	})
	return err
}

//go:build !windows
// +build !windows

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

package console

import (
	"os"
	"sync"

	"github.com/moby/term"
)

// FromFile returns a Console from the provided file
func FromFile(f *os.File) (Console, error) {
	if !term.IsTerminal(f.Fd()) {
		return nil, ErrNotAConsole
	}
	return &console{f: f}, nil
}

type console struct {
	f     *os.File
	mu    sync.Mutex
	state *term.State
}

func (c *console) Read(p []byte) (n int, err error) {
	return c.f.Read(p)
}

func (c *console) Write(p []byte) (n int, err error) {
	return c.f.Write(p)
}

func (c *console) Close() error {
	return c.f.Close()
}

func (c *console) Fd() uintptr {
	return c.f.Fd()
}

func (c *console) Name() string {
	return c.f.Name()
}

func (c *console) SetRaw() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state, err = term.SetRawTerminal(c.f.Fd())
	return err
}

func (c *console) DisableEcho() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return term.DisableEcho(c.f.Fd(), c.state)
}

func (c *console) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return term.RestoreTerminal(c.f.Fd(), c.state)
}

func (c *console) Size() (WinSize, error) {
	ws, err := term.GetWinsize(c.f.Fd())
	if err != nil {
		return WinSize{}, err
	}
	return WinSize{
		Height: ws.Height,
		Width:  ws.Width,
	}, nil
}

func (c *console) Resize(size WinSize) error {
	return term.SetWinsize(c.f.Fd(), &term.Winsize{
		Height: size.Height,
		Width:  size.Width,
	})
}

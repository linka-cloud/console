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
	"errors"
	"io"
	"os"
	"sync"

	"github.com/moby/term"
)

var (
	ErrNotAConsole = errors.New("provided file is not a console")
	ErrUnsupported = errors.New("unsupported operation")
)

type File interface {
	io.ReadWriteCloser

	// Fd returns its file descriptor
	Fd() uintptr
	// Name returns its file name
	Name() string
}

// WinSize specifies the window size of the console
type WinSize struct {
	// Height of the console
	Height uint16
	// Width of the console
	Width uint16
	x     uint16
	y     uint16
}

type Console interface {
	File

	// Resize resizes the console to the provided window size
	Resize(WinSize) error
	// SetRaw sets the console in raw mode
	SetRaw() error
	// DisableEcho disables echo on the console
	DisableEcho() error
	// Reset restores the console to its orignal state
	Reset() error
	// Size returns the window size of the console
	Size() (WinSize, error)
}

// Current returns the current process' console
func Current() (c Console) {
	var err error
	// Usually all three streams (stdin, stdout, and stderr)
	// are open to the same console, but some might be redirected,
	// so try all three.
	for _, s := range []*os.File{os.Stderr, os.Stdout, os.Stdin} {
		if c, err = FromFile(s); err == nil {
			return c
		}
	}
	// One of the std streams should always be a console
	// for the design of this function.
	panic(err)
}

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

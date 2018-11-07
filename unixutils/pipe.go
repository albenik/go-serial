//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

// +build linux darwin freebsd openbsd

package unixutils

import (
	"syscall"
)

// Pipe represents a unix-pipe
type Pipe struct {
	r int
	w int
}

// Open creates a new pipe
func (p *Pipe) Open() error {
	fds := []int{0, 0}
	if err := syscall.Pipe(fds); err != nil {
		return err
	}
	p.r = fds[0]
	p.w = fds[1]
	return nil
}

// ReadFD returns the file handle for the read side of the pipe.
func (p *Pipe) ReadFD() int {
	return p.r
}

// WriteFD returns the flie handle for the write side of the pipe.
func (p *Pipe) WriteFD() int {
	return p.w
}

// Write to the pipe the content of data. Returns the numbre of bytes written.
func (p *Pipe) Write(data []byte) (int, error) {
	return syscall.Write(p.w, data)
}

// Read from the pipe into the data array. Returns the number of bytes read.
func (p *Pipe) Read(data []byte) (int, error) {
	return syscall.Read(p.r, data)
}

// Close the pipe
func (p *Pipe) Close() error {
	err1 := syscall.Close(p.r)
	err2 := syscall.Close(p.w)
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

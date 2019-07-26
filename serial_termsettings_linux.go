//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

// +build linux

package serial

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func (p *port) retrieveTermSettings() (*settings, error) {
	s := &settings{termios: new(unix.Termios)}

	if err := ioctl(p.handle, unix.TCGETS, uintptr(unsafe.Pointer(s.termios))); err != nil {
		return nil, newOSError(err)
	}

	if s.termios.Cflag&unix.BOTHER == unix.BOTHER {
		if err := ioctl(p.handle, unix.TCGETS2, uintptr(unsafe.Pointer(s.termios))); err != nil {
			return nil, newOSError(err)
		}
	}

	return s, nil
}

func (p *port) applyTermSettings(s *settings) error {
	req := uint64(unix.TCSETS)

	if s.termios.Cflag&unix.BOTHER == unix.BOTHER {
		req = unix.TCSETS2
	}

	if err := ioctl(p.handle, req, uintptr(unsafe.Pointer(s.termios))); err != nil {
		return newOSError(err)
	}
	return nil
}

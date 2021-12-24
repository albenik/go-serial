//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

//go:build darwin
// +build darwin

package serial

// #include <sys/ioctl.h>
// #include <IOKit/serial/ioss.h>
import "C"

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func (p *Port) retrieveTermSettings() (*settings, error) {
	s := &settings{termios: new(unix.Termios), specificBaudrate: 0}

	var err error
	if s.termios, err = unix.IoctlGetTermios(p.internal.handle, unix.TIOCGETA); err != nil {
		return nil, newPortOSError(err)
	}

	speed := C.cfgetispeed((*C.struct_termios)(unsafe.Pointer(s.termios)))
	s.specificBaudrate = int(speed)

	return s, nil
}

func (p *Port) applyTermSettings(s *settings) error {
	if err := unix.IoctlSetTermios(p.internal.handle, unix.TIOCSETA, s.termios); err != nil {
		return newPortOSError(err)
	}

	speed := s.specificBaudrate
	if err := unix.IoctlSetPointerInt(p.internal.handle, C.IOSSIOSPEED, speed); err != nil {
		return newPortOSError(err)
	}

	return nil
}

//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

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
	var err error
	s := &settings{termios: new(unix.Termios), specificBaudrate: 0}

	s.termios, err = unix.IoctlGetTermios(p.internal.handle, unix.TIOCGETA)
	if err != nil {
		return nil, newOSError(err)
	}

	speed := C.cfgetispeed((*C.struct_termios)(unsafe.Pointer(s.termios)))
	s.specificBaudrate = int(speed)

	return s, nil
}

func (p *Port) applyTermSettings(s *settings) error {
	if err := unix.IoctlSetTermios(p.internal.handle, unix.TIOCSETA, s.termios); err != nil {
		return newOSError(err)
	}

	if err := unix.IoctlSetInt(p.internal.handle, C.IOSSIOSPEED, s.specificBaudrate); err != nil {
		return newOSError(err)
	}

	return nil
}

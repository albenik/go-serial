//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

// +build darwin freebsd openbsd

package serial

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func (p *Port) retrieveTermSettings() (*settings, error) {
	s := &settings{termios: new(unix.Termios)}
	if err := ioctl(p.internal.handle, ioctlTcgetattr, uintptr(unsafe.Pointer(s.termios))); err != nil {
		return nil, newOSError(err)
	}
	return s, nil
}

func (p *Port) applyTermSettings(s *settings) error {
	if err := ioctl(p.internal.handle, ioctlTcsetattr, uintptr(unsafe.Pointer(s.termios))); err != nil {
		return newOSError(err)
	}
	return nil
}

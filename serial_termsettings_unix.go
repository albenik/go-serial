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

func (p *port) retrieveTermSettings() (*unix.Termios, error) {
	settings := new(unix.Termios)
	if err := ioctl(p.handle, ioctlTcgetattr, uintptr(unsafe.Pointer(settings))); err != nil {
		return nil, newOSError(err)
	}
	return settings, nil
}

func (p *port) applyTermSettings(settings *unix.Termios) error {
	if err := ioctl(p.handle, ioctlTcsetattr, uintptr(unsafe.Pointer(settings))); err != nil {
		return newOSError(err)
	}
	return nil
}

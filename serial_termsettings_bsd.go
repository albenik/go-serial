//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

//go:build freebsd || openbsd

package serial

import (
	"golang.org/x/sys/unix"
)

func (p *Port) retrieveTermSettings() (*settings, error) {
	var err error
	s := &settings{termios: new(unix.Termios)}

	if s.termios, err = unix.IoctlGetTermios(p.internal.handle, unix.TIOCGETA); err != nil {
		return nil, newPortOSError(err)
	}

	return s, nil
}

func (p *Port) applyTermSettings(s *settings) error {
	if err := unix.IoctlSetTermios(p.internal.handle, unix.TIOCSETA, s.termios); err != nil {
		return newPortOSError(err)
	}

	return nil
}

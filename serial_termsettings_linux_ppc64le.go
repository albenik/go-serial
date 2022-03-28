//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// linux/ppc64le specific implementation, that does not use TCGETS2 or TCSETS2,
// as they are not supported by golang v1.18 for that platform circa 2022-03.
// Code is identical to android implementation.

//go:build linux && !android && ppc64le
// +build linux,!android,ppc64le

package serial

import (
	"golang.org/x/sys/unix"
)

func (p *Port) retrieveTermSettings() (s *settings, err error) {
	s = &settings{
		termios: new(unix.Termios),
	}

	if s.termios, err = unix.IoctlGetTermios(p.internal.handle, unix.TCGETS); err != nil {
		return nil, newPortOSError(err)
	}

	return s, nil
}

func (p *Port) applyTermSettings(s *settings) error {
	if err := unix.IoctlSetTermios(p.internal.handle, unix.TCSETS, s.termios); err != nil {
		return newPortOSError(err)
	}

	return nil
}

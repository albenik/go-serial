//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

// +build linux

package serial

import (
	"golang.org/x/sys/unix"
)

func (s *settings) setBaudrate(r int) error {
	if rate, ok := baudrateMap[r]; ok {
		// clear all standard baudrate bits
		for _, b := range baudrateMap {
			s.termios.Cflag &^= b
		}
		// set selected baudrate bit
		s.termios.Cflag |= rate
		s.termios.Ispeed = rate
		s.termios.Ospeed = rate
	} else {
		s.termios.Cflag &^= unix.CBAUD
		s.termios.Cflag |= unix.BOTHER
		s.termios.Ispeed = uint32(r)
		s.termios.Ospeed = uint32(r)
	}
	return nil
}

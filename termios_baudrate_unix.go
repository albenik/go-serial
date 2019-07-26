//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

// +build darwin freebsd openbsd

package serial

func (s *settings) setBaudrate(speed int) error {
	baudrate, ok := baudrateMap[speed]
	if !ok {
		return &PortError{code: InvalidSpeed}
	}
	// revert old baudrate
	for _, rate := range baudrateMap {
		s.termios.Cflag &^= rate
	}
	// set new baudrate
	s.termios.Cflag |= baudrate
	s.termios.Ispeed = toTermiosSpeedType(baudrate)
	s.termios.Ospeed = toTermiosSpeedType(baudrate)
	return nil
}

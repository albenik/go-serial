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

func setTermSettingsBaudrate(speed int, settings *unix.Termios) error {
	if rate, ok := baudrateMap[speed]; ok {
		// clear all standard baudrate bits
		for _, b := range baudrateMap {
			settings.Cflag &^= b
		}
		// set selected baudrate bit
		settings.Cflag |= rate
		settings.Ispeed = rate
		settings.Ospeed = rate
	} else {
		settings.Cflag &^= unix.CBAUD
		settings.Cflag |= unix.BOTHER
		settings.Ispeed = uint32(speed)
		settings.Ospeed = uint32(speed)
	}
	return nil
}

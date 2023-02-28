//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

//go:build darwin

package serial

import (
	"golang.org/x/sys/unix"
)

type settings struct {
	termios          *unix.Termios
	specificBaudrate int
}

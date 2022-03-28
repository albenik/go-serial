//
// Copyright 2014-2018 Cristian Maglie. All rights reserved.
// Copyright 2019-2022 Veniamin Albaev <albenik@gmail.com>.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

//go:build (linux && !android) || darwin || freebsd || openbsd
// +build linux,!android darwin freebsd openbsd

package serial

import (
	"go.uber.org/multierr"
	"golang.org/x/sys/unix"
)

func accquireExclusiveAccess(h int) error {
	return unix.IoctlSetInt(h, unix.TIOCEXCL, 0)
}

func (p *Port) closeAndReturnError(code PortErrorCode, err error) *PortError {
	return &PortError{
		code: code,
		wrapped: multierr.Combine(
			err,
			unix.IoctlSetInt(p.internal.handle, unix.TIOCNXCL, 0),
			unix.Close(p.internal.handle),
		),
	}
}

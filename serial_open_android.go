//
// Copyright 2014-2018 Cristian Maglie. All rights reserved.
// Copyright 2019 Veniamin Albaev <albenik@gmail.com>
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

//go:build android
// +build android

package serial

import (
	"go.uber.org/multierr"
	"golang.org/x/sys/unix"
)

func accquireExclusiveAccess(_ int) error {
	return nil
}

func (p *Port) closeAndReturnError(code PortErrorCode, err error) *PortError {
	return &PortError{
		code: code,
		wrapped: multierr.Combine(
			err,
			unix.Close(p.internal.handle),
		),
	}
}

//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package serial

import "golang.org/x/sys/unix"

const (
	devicesBasePath = "/dev"
	regexFilter     = "^(cu|tty)\\..*"

	ioctlTcflsh = unix.TIOCFLUSH

	tcCMSPAR uint64 = 0 // may be CMSPAR or PAREXT
	tcIUCLC  uint64 = 0

	tcCCTS_OFLOW uint64 = 0x00010000 //nolint:revive,stylecheck
	tcCRTS_IFLOW uint64 = 0x00020000 //nolint:revive,stylecheck
	tcCRTSCTS           = tcCCTS_OFLOW | tcCRTS_IFLOW
)

var databitsMap = map[int]uint64{
	0: unix.CS8, // Default to 8 bits
	5: unix.CS5,
	6: unix.CS6,
	7: unix.CS7,
	8: unix.CS8,
}

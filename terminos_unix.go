//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

// +build linux darwin freebsd openbsd

package serial

import (
	"golang.org/x/sys/unix"
)

func setTermSettingsBaudrate(speed int, settings *unix.Termios) error {
	baudrate, ok := baudrateMap[speed]
	if !ok {
		return &PortError{code: InvalidSpeed}
	}
	// revert old baudrate
	for _, rate := range baudrateMap {
		settings.Cflag &^= rate
	}
	// set new baudrate
	settings.Cflag |= baudrate
	settings.Ispeed = toTermiosSpeedType(baudrate)
	settings.Ospeed = toTermiosSpeedType(baudrate)
	return nil
}

func setTermSettingsParity(parity Parity, settings *unix.Termios) error {
	switch parity {
	case NoParity:
		settings.Cflag &^= unix.PARENB
		settings.Cflag &^= unix.PARODD
		settings.Cflag &^= tcCMSPAR
		settings.Iflag &^= unix.INPCK
	case OddParity:
		settings.Cflag |= unix.PARENB
		settings.Cflag |= unix.PARODD
		settings.Cflag &^= tcCMSPAR
		settings.Iflag |= unix.INPCK
	case EvenParity:
		settings.Cflag |= unix.PARENB
		settings.Cflag &^= unix.PARODD
		settings.Cflag &^= tcCMSPAR
		settings.Iflag |= unix.INPCK
	case MarkParity:
		if tcCMSPAR == 0 {
			return &PortError{code: InvalidParity}
		}
		settings.Cflag |= unix.PARENB
		settings.Cflag |= unix.PARODD
		settings.Cflag |= tcCMSPAR
		settings.Iflag |= unix.INPCK
	case SpaceParity:
		if tcCMSPAR == 0 {
			return &PortError{code: InvalidParity}
		}
		settings.Cflag |= unix.PARENB
		settings.Cflag &^= unix.PARODD
		settings.Cflag |= tcCMSPAR
		settings.Iflag |= unix.INPCK
	default:
		return &PortError{code: InvalidParity}
	}
	return nil
}

func setTermSettingsDataBits(bits int, settings *unix.Termios) error {
	databits, ok := databitsMap[bits]
	if !ok {
		return &PortError{code: InvalidDataBits}
	}
	// Remove previous databits setting
	settings.Cflag &^= unix.CSIZE
	// Set requested databits
	settings.Cflag |= databits
	return nil
}

func setTermSettingsStopBits(bits StopBits, settings *unix.Termios) error {
	switch bits {
	case OneStopBit:
		settings.Cflag &^= unix.CSTOPB
	case OnePointFiveStopBits:
		return &PortError{code: InvalidStopBits}
	case TwoStopBits:
		settings.Cflag |= unix.CSTOPB
	default:
		return &PortError{code: InvalidStopBits}
	}
	return nil
}

func setTermSettingsCtsRts(settings *unix.Termios, enable bool) {
	if enable {
		settings.Cflag |= tcCRTSCTS
	} else {
		settings.Cflag &^= tcCRTSCTS
	}
}

func setTermSettingRawMode(settings *unix.Termios) {
	// Set local mode
	settings.Cflag |= unix.CREAD
	settings.Cflag |= unix.CLOCAL

	// Set raw mode
	settings.Lflag &^= unix.ICANON
	settings.Lflag &^= unix.ECHO
	settings.Lflag &^= unix.ECHOE
	settings.Lflag &^= unix.ECHOK
	settings.Lflag &^= unix.ECHONL
	settings.Lflag &^= unix.ECHOCTL
	settings.Lflag &^= unix.ECHOPRT
	settings.Lflag &^= unix.ECHOKE
	settings.Lflag &^= unix.ISIG
	settings.Lflag &^= unix.IEXTEN

	settings.Iflag &^= unix.IXON
	settings.Iflag &^= unix.IXOFF
	settings.Iflag &^= unix.IXANY
	settings.Iflag &^= unix.INPCK
	settings.Iflag &^= unix.IGNPAR
	settings.Iflag &^= unix.PARMRK
	settings.Iflag &^= unix.ISTRIP
	settings.Iflag &^= unix.IGNBRK
	settings.Iflag &^= unix.BRKINT
	settings.Iflag &^= unix.INLCR
	settings.Iflag &^= unix.IGNCR
	settings.Iflag &^= unix.ICRNL
	settings.Iflag &^= tcIUCLC

	settings.Oflag &^= unix.OPOST

	// Block reads until at least one char is available (no timeout)
	settings.Cc[unix.VMIN] = 1
	settings.Cc[unix.VTIME] = 0
}

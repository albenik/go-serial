//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

//go:build linux || darwin || freebsd || openbsd
// +build linux darwin freebsd openbsd

package serial

import (
	"golang.org/x/sys/unix"
)

func (s *settings) setParity(parity Parity) error {
	switch parity {
	case NoParity:
		s.termios.Cflag &^= unix.PARENB
		s.termios.Cflag &^= unix.PARODD
		s.termios.Cflag &^= tcCMSPAR
		s.termios.Iflag &^= unix.INPCK
	case OddParity:
		s.termios.Cflag |= unix.PARENB
		s.termios.Cflag |= unix.PARODD
		s.termios.Cflag &^= tcCMSPAR
		s.termios.Iflag |= unix.INPCK
	case EvenParity:
		s.termios.Cflag |= unix.PARENB
		s.termios.Cflag &^= unix.PARODD
		s.termios.Cflag &^= tcCMSPAR
		s.termios.Iflag |= unix.INPCK
	case MarkParity:
		if tcCMSPAR == 0 {
			return &PortError{code: InvalidParity}
		}
		s.termios.Cflag |= unix.PARENB
		s.termios.Cflag |= unix.PARODD
		s.termios.Cflag |= tcCMSPAR
		s.termios.Iflag |= unix.INPCK
	case SpaceParity:
		if tcCMSPAR == 0 {
			return &PortError{code: InvalidParity}
		}
		s.termios.Cflag |= unix.PARENB
		s.termios.Cflag &^= unix.PARODD
		s.termios.Cflag |= tcCMSPAR
		s.termios.Iflag |= unix.INPCK
	default:
		return &PortError{code: InvalidParity}
	}
	return nil
}

func (s *settings) setDataBits(bits int) error {
	databits, ok := databitsMap[bits]
	if !ok {
		return &PortError{code: InvalidDataBits}
	}
	// Remove previous databits setting
	s.termios.Cflag &^= unix.CSIZE
	// Set requested databits
	s.termios.Cflag |= databits
	return nil
}

func (s *settings) setStopBits(bits StopBits) error {
	switch bits {
	case OneStopBit:
		s.termios.Cflag &^= unix.CSTOPB
	case OnePointFiveStopBits:
		return &PortError{code: InvalidStopBits}
	case TwoStopBits:
		s.termios.Cflag |= unix.CSTOPB
	default:
		return &PortError{code: InvalidStopBits}
	}
	return nil
}

func (s *settings) setCtsRts(enable bool) {
	if enable {
		s.termios.Cflag |= tcCRTSCTS
	} else {
		s.termios.Cflag &^= tcCRTSCTS
	}
}

func (s *settings) setRawMode(hupcl bool) {
	// Set local mode
	s.termios.Cflag |= unix.CREAD
	s.termios.Cflag |= unix.CLOCAL
	if hupcl {
		s.termios.Cflag |= unix.HUPCL
	} else {
		s.termios.Cflag &^= unix.HUPCL
	}

	// Set raw mode
	s.termios.Lflag &^= unix.ICANON
	s.termios.Lflag &^= unix.ECHO
	s.termios.Lflag &^= unix.ECHOE
	s.termios.Lflag &^= unix.ECHOK
	s.termios.Lflag &^= unix.ECHONL
	s.termios.Lflag &^= unix.ECHOCTL
	s.termios.Lflag &^= unix.ECHOPRT
	s.termios.Lflag &^= unix.ECHOKE
	s.termios.Lflag &^= unix.ISIG
	s.termios.Lflag &^= unix.IEXTEN

	s.termios.Iflag &^= unix.IXON
	s.termios.Iflag &^= unix.IXOFF
	s.termios.Iflag &^= unix.IXANY
	s.termios.Iflag &^= unix.INPCK
	s.termios.Iflag &^= unix.IGNPAR
	s.termios.Iflag &^= unix.PARMRK
	s.termios.Iflag &^= unix.ISTRIP
	s.termios.Iflag &^= unix.IGNBRK
	s.termios.Iflag &^= unix.BRKINT
	s.termios.Iflag &^= unix.INLCR
	s.termios.Iflag &^= unix.IGNCR
	s.termios.Iflag &^= unix.ICRNL
	s.termios.Iflag &^= tcIUCLC

	s.termios.Oflag &^= unix.OPOST

	// Block reads until at least one char is available (no timeout)
	s.termios.Cc[unix.VMIN] = 1
	s.termios.Cc[unix.VTIME] = 0
}

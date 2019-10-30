//
// Copyright 2014-2018 Cristian Maglie.
// Copyright 2019 Veniamin Albaev <albenik@gmail.com>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package serial

import (
	"os"
)

//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zsyscall_windows.go syscall_windows.go

const (
	// NoParity disable parity control (default)
	NoParity Parity = iota
	// OddParity enable odd-parity check
	OddParity
	// EvenParity enable even-parity check
	EvenParity
	// MarkParity enable mark-parity (always 1) check
	MarkParity
	// SpaceParity enable space-parity (always 0) check
	SpaceParity

	// OneStopBit sets 1 stop bit (default)
	OneStopBit StopBits = iota
	// OnePointFiveStopBits sets 1.5 stop bits
	OnePointFiveStopBits
	// TwoStopBits sets 2 stop bits
	TwoStopBits
)

// StopBits describe a serial port stop bits setting
type StopBits int

// Parity describes a serial port parity setting
type Parity int

// ModemStatusBits contains all the modem status bits for a serial port (CTS, DSR, etc...).
// It can be retrieved with the Port.GetModemStatusBits() method.
type ModemStatusBits struct {
	CTS bool // ClearToSend status
	DSR bool // DataSetReady status
	RI  bool // RingIndicator status
	DCD bool // DataCarrierDetect status
}

// Port is the interface for a serial Port
type Port struct {
	name     string
	opened   bool
	baudRate int      // The serial port bitrate (aka Baudrate)
	dataBits int      // Size of the character (must be 5, 6, 7 or 8)
	parity   Parity   // Parity (see Parity type for more info)
	stopBits StopBits // Stop bits (see StopBits type for more info)
	hupcl    bool     // Lower DTR line on close (hang up)

	internal *port // os specific (implementation like os.File)
}

func (p *Port) String() string {
	if p == nil {
		return "Error: <nil> port instance"
	}
	return p.name
}

func (p *Port) checkValid() error {
	if p == nil || p.internal == nil || !isHandleValid(p.internal.handle) {
		return &PortError{code: PortClosed, causedBy: os.ErrInvalid}
	}
	if !p.opened {
		return &PortError{code: PortClosed}
	}
	return nil
}

func newWithDefaults(n string, p *port) *Port {
	return &Port{
		name:     n,
		opened:   true,
		baudRate: 9600,
		dataBits: 8,
		parity:   NoParity,
		stopBits: OneStopBit,
		hupcl:    false,
		internal: p,
	}
}

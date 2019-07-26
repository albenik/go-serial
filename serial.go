//
// Copyright 2014-2018 Cristian Maglie.
// Copyright 2019 Veniamin Albaev <albenik@gmail.com>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package serial

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

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
	*port // os specific (implementation like os.File)

	name     string
	baudRate int      // The serial port bitrate (aka Baudrate)
	dataBits int      // Size of the character (must be 5, 6, 7 or 8)
	parity   Parity   // Parity (see Parity type for more info)
	stopBits StopBits // Stop bits (see StopBits type for more info)
	hupcl    bool     // Lower DTR line on close (hang down)
}

func newWithDefaults(n string, p *port) *Port {
	return &Port{
		port:     p,
		name:     n,
		baudRate: 9600,
		dataBits: 8,
		parity:   NoParity,
		stopBits: OneStopBit,
		hupcl:    false,
	}
}

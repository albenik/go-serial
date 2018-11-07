//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

// +build linux darwin freebsd openbsd

package serial

import (
	"io/ioutil"
	"regexp"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/albenik/go-serial/unixutils"
)

type unixPort struct {
	name   string
	handle int
	opened bool

	firstByteTimeout bool
	readTimeout      int
	writeTimeout     int

	closeSignal *unixutils.Pipe // TODO async port close implementation not clear
}

func (port *unixPort) String() string {
	return port.name
}

func (port *unixPort) Close() error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}

	err := func() error {
		// Send close signal to all pending reads (if any) and close signaling pipe
		if _, err := port.closeSignal.Write([]byte{0}); err != nil {
			return err
		}
		if err := port.closeSignal.Close(); err != nil {
			return err
		}
		// Release exclusive access
		if err := ioctl(port.handle, unix.TIOCNXCL, 0); err != nil {
			return err
		}
		if err := unix.Close(port.handle); err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return &PortError{code: OsError, causedBy: err}
	}
	return nil
}

func (port *unixPort) ReadyToRead() (uint32, error) {
	if !port.opened {
		return 0, &PortError{code: PortClosed}
	}

	var n uint32
	if err := ioctl(port.handle, FIONREAD, uintptr(unsafe.Pointer(&n))); err != nil {
		return 0, &PortError{code: OsError, causedBy: err}
	}
	return n, nil
}

func (port *unixPort) Read(p []byte) (int, error) {
	if !port.opened {
		return 0, &PortError{code: PortClosed}
	}

	size, read := len(p), 0
	fds := unixutils.NewFDSet(port.handle, port.closeSignal.ReadFD())
	buf := make([]byte, size)

	now := time.Now()
	deadline := now.Add(time.Duration(port.readTimeout) * time.Millisecond)

	for read < size {
		res, err := unixutils.Select(fds, nil, fds, deadline.Sub(now))
		if err != nil {
			return read, err
		}
		if res.IsReadable(port.closeSignal.ReadFD()) {
			return read, &PortError{code: PortClosed}
		}
		if !res.IsReadable(port.handle) {
			return read, nil
		}

		n, err := unix.Read(port.handle, buf[read:])
		if err != nil {
			return read, err
		}
		// read should always return some data as select reported, it was ready to read when we got to this point.
		if n == 0 {
			return read, &PortError{code: ReadFailed}
		}

		copy(p[read:], buf[read:read+n])
		read += n

		now = time.Now()
		if !now.Before(deadline) || port.firstByteTimeout {
			return read, nil
		}
	}
	return read, nil
}

func (port *unixPort) Write(p []byte) (int, error) {
	if !port.opened {
		return 0, &PortError{code: PortClosed}
	}

	size, written := len(p), 0
	fds := unixutils.NewFDSet(port.handle)
	clFds := unixutils.NewFDSet(port.closeSignal.ReadFD())

	deadline := time.Now().Add(time.Duration(port.writeTimeout) * time.Millisecond)

	for written < size {
		n, err := unix.Write(port.handle, p[written:])
		if err != nil {
			return written, err
		}
		if port.writeTimeout == 0 {
			return n, nil
		}
		written += n
		now := time.Now()
		if port.writeTimeout > 0 && !now.Before(deadline) {
			return written, nil
		}
		res, err := unixutils.Select(clFds, fds, fds, deadline.Sub(now))
		if err != nil {
			return written, err
		}
		if res.IsReadable(port.closeSignal.ReadFD()) {
			return written, &PortError{code: PortClosed}
		}
		if !res.IsWritable(port.handle) {
			return written, &PortError{code: WriteFailed}
		}
	}
	return written, nil
}

func (port *unixPort) ResetInputBuffer() error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}
	return ioctl(port.handle, ioctlTcflsh, unix.TCIFLUSH)
}

func (port *unixPort) ResetOutputBuffer() error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}
	return ioctl(port.handle, ioctlTcflsh, unix.TCOFLUSH)
}

func (port *unixPort) SetMode(mode *Mode) error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}

	settings, err := port.getTermSettings()
	if err != nil {
		return err
	}
	if err := setTermSettingsBaudrate(mode.BaudRate, settings); err != nil {
		return err
	}
	if err := setTermSettingsParity(mode.Parity, settings); err != nil {
		return err
	}
	if err := setTermSettingsDataBits(mode.DataBits, settings); err != nil {
		return err
	}
	if err := setTermSettingsStopBits(mode.StopBits, settings); err != nil {
		return err
	}
	return port.setTermSettings(settings)
}

func (port *unixPort) SetDTR(dtr bool) error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}

	status, err := port.getModemBitsStatus()
	if err != nil {
		return err
	}
	if dtr {
		status |= unix.TIOCM_DTR
	} else {
		status &^= unix.TIOCM_DTR
	}
	return port.setModemBitsStatus(status)
}

func (port *unixPort) SetRTS(rts bool) error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}

	status, err := port.getModemBitsStatus()
	if err != nil {
		return err
	}
	if rts {
		status |= unix.TIOCM_RTS
	} else {
		status &^= unix.TIOCM_RTS
	}
	return port.setModemBitsStatus(status)
}

func (port *unixPort) SetReadTimeout(t int) error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}

	port.firstByteTimeout = false
	port.readTimeout = t
	return nil // timeout is done via select
}

func (port *unixPort) SetReadTimeoutEx(t, i uint32) error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}

	settings, err := port.getTermSettings()
	if err != nil {
		return err
	}

	vtime := t / 100 // VTIME tenths of a second elapses between bytes
	if vtime > 255 || vtime*100 != t {
		return &PortError{code: InvalidTimeoutValue}
	}
	if vtime > 0 {
		settings.Cc[unix.VMIN] = 1
		settings.Cc[unix.VTIME] = uint8(t)
	} else {
		settings.Cc[unix.VMIN] = 0
		settings.Cc[unix.VTIME] = 0
	}

	if err = port.setTermSettings(settings); err != nil {
		return &PortError{code: OsError, causedBy: err}
	}

	port.firstByteTimeout = false
	port.readTimeout = int(t)
	return nil
}

func (port *unixPort) SetFirstByteReadTimeout(t uint32) error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}

	if t > 0 && t < 0xFFFFFFFF {
		port.firstByteTimeout = true
		port.readTimeout = int(t)
		return nil
	} else {
		return &PortError{code: InvalidTimeoutValue}
	}
}

func (port *unixPort) SetWriteTimeout(t int) error {
	if !port.opened {
		return &PortError{code: PortClosed}
	}

	port.writeTimeout = t
	return nil // timeout is done via select
}

func (port *unixPort) GetModemStatusBits() (*ModemStatusBits, error) {
	if !port.opened {
		return nil, &PortError{code: PortClosed}
	}

	status, err := port.getModemBitsStatus()
	if err != nil {
		return nil, err
	}
	return &ModemStatusBits{
		CTS: (status & unix.TIOCM_CTS) != 0,
		DCD: (status & unix.TIOCM_CD) != 0,
		DSR: (status & unix.TIOCM_DSR) != 0,
		RI:  (status & unix.TIOCM_RI) != 0,
	}, nil
}

func nativeOpen(portName string, mode *Mode) (*unixPort, error) {
	h, err := unix.Open(portName, unix.O_RDWR|unix.O_NOCTTY|unix.O_NDELAY, 0)
	if err != nil {
		switch err {
		case unix.EBUSY:
			return nil, &PortError{code: PortBusy}
		case unix.EACCES:
			return nil, &PortError{code: PermissionDenied}
		}
		return nil, err
	}
	port := &unixPort{
		name:   portName,
		handle: h,
		opened: true,

		firstByteTimeout: true,
		readTimeout:      1000, // Backward compatible default value
		writeTimeout:     0,
	}

	// Setup serial port
	if port.SetMode(mode) != nil {
		port.Close()
		return nil, &PortError{code: InvalidSerialPort}
	}

	settings, err := port.getTermSettings()
	if err != nil {
		port.Close()
		return nil, &PortError{code: InvalidSerialPort}
	}

	// Set raw mode
	setRawMode(settings)

	// Explicitly disable RTS/CTS flow control
	setTermSettingsCtsRts(false, settings)

	if port.setTermSettings(settings) != nil {
		port.Close()
		return nil, &PortError{code: InvalidSerialPort}
	}

	if err = unix.SetNonblock(h, false); err != nil {
		return nil, &PortError{code: OsError, causedBy: err}
	}
	// Accquire exclusive access
	if err = ioctl(port.handle, unix.TIOCEXCL, 0); err != nil {
		return nil, &PortError{code: OsError, causedBy: err}
	}

	// This pipe is used as a signal to cancel blocking Read
	pipe := &unixutils.Pipe{}
	if err := pipe.Open(); err != nil {
		port.Close()
		return nil, &PortError{code: InvalidSerialPort, causedBy: err}
	}
	port.closeSignal = pipe

	return port, nil
}

func nativeGetPortsList() ([]string, error) {
	files, err := ioutil.ReadDir(devFolder)
	if err != nil {
		return nil, err
	}

	ports := make([]string, 0, len(files))
	for _, f := range files {
		// Skip folders
		if f.IsDir() {
			continue
		}

		// Keep only devices with the correct name
		match, err := regexp.MatchString(regexFilter, f.Name())
		if err != nil {
			return nil, err
		}
		if !match {
			continue
		}

		portName := devFolder + "/" + f.Name()

		// Check if serial port is real or is a placeholder serial port "ttySxx"
		if strings.HasPrefix(f.Name(), "ttyS") {
			port, err := nativeOpen(portName, &Mode{})
			if err != nil {
				serr, ok := err.(*PortError)
				if ok && serr.Code() == InvalidSerialPort {
					continue
				}
			} else {
				port.Close()
			}
		}

		// Save serial port in the resulting list
		ports = append(ports, portName)
	}

	return ports, nil
}

// termios manipulation functions

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

func setTermSettingsCtsRts(enable bool, settings *unix.Termios) {
	if enable {
		settings.Cflag |= tcCRTSCTS
	} else {
		settings.Cflag &^= tcCRTSCTS
	}
}

func setRawMode(settings *unix.Termios) {
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

// native syscall wrapper functions

func (port *unixPort) getTermSettings() (*unix.Termios, error) {
	settings := new(unix.Termios)
	err := ioctl(port.handle, ioctlTcgetattr, uintptr(unsafe.Pointer(settings)))
	return settings, err
}

func (port *unixPort) setTermSettings(settings *unix.Termios) error {
	return ioctl(port.handle, ioctlTcsetattr, uintptr(unsafe.Pointer(settings)))
}

func (port *unixPort) getModemBitsStatus() (int, error) {
	var status int
	err := ioctl(port.handle, unix.TIOCMGET, uintptr(unsafe.Pointer(&status)))
	return status, err
}

func (port *unixPort) setModemBitsStatus(status int) error {
	return ioctl(port.handle, unix.TIOCMSET, uintptr(unsafe.Pointer(&status)))
}

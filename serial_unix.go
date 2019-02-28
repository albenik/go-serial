//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

// +build linux darwin freebsd openbsd

package serial

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"go.uber.org/multierr"
	"golang.org/x/sys/unix"

	"github.com/albenik/go-serial/unixutils"
)

var zeroByte = []byte{0}

type unixPort struct {
	name   string
	handle int
	opened bool

	firstByteTimeout bool
	readTimeout      int
	writeTimeout     int

	closePipeR int
	closePipeW int
}

func getPortsList() ([]string, error) {
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
			port, err := openPort(portName, &Mode{})
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

func openPort(portName string, mode *Mode) (*unixPort, error) {
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

	// Accquire exclusive access
	if err = ioctl(h, unix.TIOCEXCL, 0); err != nil {
		return nil, newOSError(multierr.Append(err, unix.Close(h)))
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
	if err := port.SetMode(mode); err != nil {
		return nil, port.closeAndReturnError(InvalidSerialPort, err)
	}

	settings, err := port.retrieveTermSettings()
	if err != nil {
		return nil, port.closeAndReturnError(InvalidSerialPort, err)
	}

	// Set raw mode
	setTermSettingRawMode(settings)

	// Explicitly disable RTS/CTS flow control
	setTermSettingsCtsRts(settings, false)

	if err = port.applyTermSettings(settings); err != nil {
		return nil, port.closeAndReturnError(InvalidSerialPort, err)
	}

	if err = unix.SetNonblock(h, false); err != nil {
		return nil, port.closeAndReturnError(OsError, err)
	}

	fds := []int{0, 0}
	if err := syscall.Pipe(fds); err != nil {
		port.Close()
		return nil, port.closeAndReturnError(OsError, err)
	}
	port.closePipeR = fds[0]
	port.closePipeW = fds[1]

	return port, nil
}

func (port *unixPort) Close() error {
	// NOT thread safe
	if err := port.checkValid(); err != nil {
		return err
	}

	port.opened = false

	// Send close signal to all pending reads (if any) and close signaling pipe
	_, err := unix.Write(port.closePipeW, zeroByte)
	err = multierr.Combine(
		err,
		unix.Close(port.closePipeW),
		unix.Close(port.closePipeR),
		ioctl(port.handle, unix.TIOCNXCL, 0),
		unix.Close(port.handle),
	)

	if err != nil {
		return newOSError(err)
	}
	return nil
}

func (port *unixPort) String() string {
	if port == nil {
		return "INVALID_NULL_PORT"
	}
	return port.name
}

func (port *unixPort) ReadyToRead() (uint32, error) {
	if err := port.checkValid(); err != nil {
		return 0, err
	}

	var n uint32
	if err := ioctl(port.handle, FIONREAD, uintptr(unsafe.Pointer(&n))); err != nil {
		return 0, newOSError(err)
	}
	return n, nil
}

func (port *unixPort) Read(p []byte) (int, error) {
	if err := port.checkValid(); err != nil {
		return 0, err
	}

	size, read := len(p), 0
	fds := unixutils.NewFDSet(port.handle, port.closePipeR)
	buf := make([]byte, size)

	now := time.Now()
	deadline := now.Add(time.Duration(port.readTimeout) * time.Millisecond)

	for read < size {
		res, err := unixutils.Select(fds, nil, fds, deadline.Sub(now))
		if err != nil {
			return read, newOSError(err)
		}

		if res.IsReadable(port.closePipeR) {
			return read, &PortError{code: PortClosed}
		}
		if !res.IsReadable(port.handle) {
			return read, nil
		}

		n, err := unix.Read(port.handle, buf[read:])
		if err != nil {
			return read, newOSError(err)
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
	if err := port.checkValid(); err != nil {
		return 0, err
	}

	size, written := len(p), 0
	fds := unixutils.NewFDSet(port.handle)
	clFds := unixutils.NewFDSet(port.closePipeR)

	deadline := time.Now().Add(time.Duration(port.writeTimeout) * time.Millisecond)

	for written < size {
		n, err := unix.Write(port.handle, p[written:])
		if err != nil {
			return written, newOSError(err)
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
			return written, newOSError(err)
		}

		if res.IsReadable(port.closePipeR) {
			return written, &PortError{code: PortClosed}
		}

		if !res.IsWritable(port.handle) {
			return written, &PortError{code: WriteFailed}
		}
	}
	return written, nil
}

func (port *unixPort) ResetInputBuffer() error {
	if err := port.checkValid(); err != nil {
		return err
	}

	if err := ioctl(port.handle, ioctlTcflsh, unix.TCIFLUSH); err != nil {
		return newOSError(err)
	}
	return nil
}

func (port *unixPort) ResetOutputBuffer() error {
	if err := port.checkValid(); err != nil {
		return err
	}

	if err := ioctl(port.handle, ioctlTcflsh, unix.TCOFLUSH); err != nil {
		return newOSError(err)
	}
	return nil
}

func (port *unixPort) SetMode(mode *Mode) error {
	if err := port.checkValid(); err != nil {
		return err
	}

	settings, err := port.retrieveTermSettings()
	if err != nil {
		return err // port.retrieveTermSettings() already returned PortError
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
	return port.applyTermSettings(settings) // already returned PortError
}

func (port *unixPort) SetDTR(dtr bool) error {
	if err := port.checkValid(); err != nil {
		return err
	}

	status, err := port.retrieveModemBitsStatus()
	if err != nil {
		return err // port.retrieveModemBitsStatus already returned PortError
	}
	if dtr {
		status |= unix.TIOCM_DTR
	} else {
		status &^= unix.TIOCM_DTR
	}
	return port.applyModemBitsStatus(status) // already returned PortError
}

func (port *unixPort) SetRTS(rts bool) error {
	if err := port.checkValid(); err != nil {
		return err
	}

	status, err := port.retrieveModemBitsStatus()
	if err != nil {
		return err // port.retrieveModemBitsStatus() already returned PortError
	}
	if rts {
		status |= unix.TIOCM_RTS
	} else {
		status &^= unix.TIOCM_RTS
	}
	return port.applyModemBitsStatus(status) // already returned PortError
}

func (port *unixPort) SetReadTimeout(t int) error {
	if err := port.checkValid(); err != nil {
		return err
	}

	port.firstByteTimeout = false
	port.readTimeout = t
	return nil // timeout is done via select
}

func (port *unixPort) SetReadTimeoutEx(t, i uint32) error {
	if err := port.checkValid(); err != nil {
		return err
	}

	settings, err := port.retrieveTermSettings()
	if err != nil {
		return err // port.retrieveTermSettings() already returned PortError
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

	if err = port.applyTermSettings(settings); err != nil {
		return err // port.applyTermSettings() already returned PortError
	}

	port.firstByteTimeout = false
	port.readTimeout = int(t)
	return nil
}

func (port *unixPort) SetFirstByteReadTimeout(t uint32) error {
	if err := port.checkValid(); err != nil {
		return err
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
	if err := port.checkValid(); err != nil {
		return err
	}

	port.writeTimeout = t
	return nil // timeout is done via select
}

func (port *unixPort) GetModemStatusBits() (*ModemStatusBits, error) {
	if err := port.checkValid(); err != nil {
		return nil, err
	}

	status, err := port.retrieveModemBitsStatus()
	if err != nil {
		return nil, err // port.retrieveModemBitsStatus() already returned PortError
	}
	return &ModemStatusBits{
		CTS: (status & unix.TIOCM_CTS) != 0,
		DCD: (status & unix.TIOCM_CD) != 0,
		DSR: (status & unix.TIOCM_DSR) != 0,
		RI:  (status & unix.TIOCM_RI) != 0,
	}, nil
}

func (port *unixPort) closeAndReturnError(code PortErrorCode, err error) *PortError {
	return &PortError{code: code, causedBy: multierr.Combine(
		err,
		ioctl(port.handle, unix.TIOCNXCL, 0),
		unix.Close(port.handle),
	)}
}

func (port *unixPort) checkValid() error {
	if port == nil {
		return &PortError{code: PortClosed, causedBy: os.ErrInvalid}
	}
	if !port.opened {
		return &PortError{code: PortClosed}
	}
	return nil
}

func (port *unixPort) retrieveModemBitsStatus() (int, error) {
	var status int
	if err := ioctl(port.handle, unix.TIOCMGET, uintptr(unsafe.Pointer(&status))); err != nil {
		return 0, newOSError(err)
	}
	return status, nil
}

func (port *unixPort) applyModemBitsStatus(status int) error {
	if err := ioctl(port.handle, unix.TIOCMSET, uintptr(unsafe.Pointer(&status))); err != nil {
		return newOSError(err)
	}
	return nil
}

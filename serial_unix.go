//
// Copyright 2014-2018 Cristian Maglie. All rights reserved.
// Copyright 2019 Veniamin Albaev <albenik@gmail.com>
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

type port struct {
	handle int
	opened bool

	firstByteTimeout bool
	readTimeout      int
	writeTimeout     int

	closePipeR int
	closePipeW int
}

func Open(name string, opts ...Option) (*Port, error) {
	h, err := unix.Open(name, unix.O_RDWR|unix.O_NOCTTY|unix.O_NDELAY, 0)
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

	port := newWithDefaults(name, &port{
		handle: h,
		opened: true,

		firstByteTimeout: true,
		readTimeout:      0,
		writeTimeout:     0,
	})

	// Setup serial port
	if err := port.Reconfigure(opts...); err != nil {
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

func (p *Port) Reconfigure(opts ...Option) error {
	for _, o := range opts {
		o(p)
	}
	return p.reconfigure()
}

func (p *Port) Close() error {
	// NOT thread safe
	if err := p.checkValid(); err != nil {
		return err
	}

	p.opened = false

	// Send close signal to all pending reads (if any) and close signaling pipe
	_, err := unix.Write(p.closePipeW, zeroByte)
	err = multierr.Combine(
		err,
		unix.Close(p.closePipeW),
		unix.Close(p.closePipeR),
		ioctl(p.handle, unix.TIOCNXCL, 0),
		unix.Close(p.handle),
	)

	if err != nil {
		return newOSError(err)
	}
	return nil
}

func (p *Port) String() string {
	if p == nil {
		return "INVALID_NULL_PORT"
	}
	return p.name
}

func (p *Port) ReadyToRead() (uint32, error) {
	if err := p.checkValid(); err != nil {
		return 0, err
	}

	var n uint32
	if err := ioctl(p.handle, FIONREAD, uintptr(unsafe.Pointer(&n))); err != nil {
		return 0, newOSError(err)
	}
	return n, nil
}

func (p *Port) Read(b []byte) (int, error) {
	if err := p.checkValid(); err != nil {
		return 0, err
	}

	size, read := len(b), 0
	fds := unixutils.NewFDSet(p.handle, p.closePipeR)
	buf := make([]byte, size)

	now := time.Now()
	deadline := now.Add(time.Duration(p.readTimeout) * time.Millisecond)

	for read < size {
		res, err := unixutils.Select(fds, nil, fds, deadline.Sub(now))
		if err != nil {
			return read, newOSError(err)
		}

		if res.IsReadable(p.closePipeR) {
			return read, &PortError{code: PortClosed}
		}
		if !res.IsReadable(p.handle) {
			return read, nil
		}

		n, err := unix.Read(p.handle, buf[read:])
		if err != nil {
			return read, newOSError(err)
		}
		// read should always return some data as select reported, it was ready to read when we got to this point.
		if n == 0 {
			return read, &PortError{code: ReadFailed}
		}

		copy(b[read:], buf[read:read+n])
		read += n

		now = time.Now()
		if !now.Before(deadline) || p.firstByteTimeout {
			return read, nil
		}
	}
	return read, nil
}

func (p *Port) Write(b []byte) (int, error) {
	if err := p.checkValid(); err != nil {
		return 0, err
	}

	size, written := len(b), 0
	fds := unixutils.NewFDSet(p.handle)
	clFds := unixutils.NewFDSet(p.closePipeR)

	deadline := time.Now().Add(time.Duration(p.writeTimeout) * time.Millisecond)

	for written < size {
		n, err := unix.Write(p.handle, b[written:])
		if err != nil {
			return written, newOSError(err)
		}

		if p.writeTimeout == 0 {
			return n, nil
		}

		written += n
		now := time.Now()
		if p.writeTimeout > 0 && !now.Before(deadline) {
			return written, nil
		}

		res, err := unixutils.Select(clFds, fds, fds, deadline.Sub(now))
		if err != nil {
			return written, newOSError(err)
		}

		if res.IsReadable(p.closePipeR) {
			return written, &PortError{code: PortClosed}
		}

		if !res.IsWritable(p.handle) {
			return written, &PortError{code: WriteFailed}
		}
	}
	return written, nil
}

func (p *Port) ResetInputBuffer() error {
	if err := p.checkValid(); err != nil {
		return err
	}

	if err := ioctl(p.handle, ioctlTcflsh, unix.TCIFLUSH); err != nil {
		return newOSError(err)
	}
	return nil
}

func (p *Port) ResetOutputBuffer() error {
	if err := p.checkValid(); err != nil {
		return err
	}

	if err := ioctl(p.handle, ioctlTcflsh, unix.TCOFLUSH); err != nil {
		return newOSError(err)
	}
	return nil
}

func (p *Port) SetDTR(dtr bool) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	status, err := p.retrieveModemBitsStatus()
	if err != nil {
		return err // port.retrieveModemBitsStatus already returned PortError
	}
	if dtr {
		status |= unix.TIOCM_DTR
	} else {
		status &^= unix.TIOCM_DTR
	}
	return p.applyModemBitsStatus(status) // already returned PortError
}

func (p *Port) SetRTS(rts bool) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	status, err := p.retrieveModemBitsStatus()
	if err != nil {
		return err // port.retrieveModemBitsStatus() already returned PortError
	}
	if rts {
		status |= unix.TIOCM_RTS
	} else {
		status &^= unix.TIOCM_RTS
	}
	return p.applyModemBitsStatus(status) // already returned PortError
}

func (p *Port) SetReadTimeout(t int) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	p.setReadTimeoutValues(t)
	return nil // timeout is done via select
}

func (p *Port) SetReadTimeoutEx(t, i uint32) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	settings, err := p.retrieveTermSettings()
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

	if err = p.applyTermSettings(settings); err != nil {
		return err // port.applyTermSettings() already returned PortError
	}

	p.firstByteTimeout = false
	p.readTimeout = int(t)
	return nil
}

func (p *Port) SetFirstByteReadTimeout(t uint32) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	if t > 0 && t < 0xFFFFFFFF {
		p.firstByteTimeout = true
		p.readTimeout = int(t)
		return nil
	} else {
		return &PortError{code: InvalidTimeoutValue}
	}
}

func (p *Port) SetWriteTimeout(t int) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	p.setWriteTimeoutValues(t)
	return nil // timeout is done via select
}

func (p *Port) GetModemStatusBits() (*ModemStatusBits, error) {
	if err := p.checkValid(); err != nil {
		return nil, err
	}

	status, err := p.retrieveModemBitsStatus()
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

func (p *Port) closeAndReturnError(code PortErrorCode, err error) *PortError {
	return &PortError{code: code, causedBy: multierr.Combine(
		err,
		ioctl(p.handle, unix.TIOCNXCL, 0),
		unix.Close(p.handle),
	)}
}

func (p *Port) checkValid() error {
	if p == nil {
		return &PortError{code: PortClosed, causedBy: os.ErrInvalid}
	}
	if !p.opened {
		return &PortError{code: PortClosed}
	}
	return nil
}

func (p *Port) setReadTimeoutValues(t int) {
	p.firstByteTimeout = false
	p.readTimeout = t
}

func (p *Port) setWriteTimeoutValues(t int) {
	p.writeTimeout = t
}

func (p *Port) retrieveModemBitsStatus() (int, error) {
	var status int
	if err := ioctl(p.handle, unix.TIOCMGET, uintptr(unsafe.Pointer(&status))); err != nil {
		return 0, newOSError(err)
	}
	return status, nil
}

func (p *Port) applyModemBitsStatus(status int) error {
	if err := ioctl(p.handle, unix.TIOCMSET, uintptr(unsafe.Pointer(&status))); err != nil {
		return newOSError(err)
	}
	return nil
}

func (p *Port) reconfigure() error {
	if err := p.checkValid(); err != nil {
		return err
	}

	settings, err := p.retrieveTermSettings()
	if err != nil {
		return err // port.retrieveTermSettings() already returned PortError
	}
	if err := setTermSettingsBaudrate(p.baudRate, settings); err != nil {
		return err
	}
	if err := setTermSettingsParity(p.parity, settings); err != nil {
		return err
	}
	if err := setTermSettingsDataBits(p.dataBits, settings); err != nil {
		return err
	}
	if err := setTermSettingsStopBits(p.stopBits, settings); err != nil {
		return err
	}
	return p.applyTermSettings(settings) // already returned PortError
}

func GetPortsList() ([]string, error) {
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

		name := devFolder + "/" + f.Name()

		// Check if serial port is real or is a placeholder serial port "ttySxx"
		if strings.HasPrefix(f.Name(), "ttyS") {
			port, err := Open(name)
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
		ports = append(ports, name)
	}

	return ports, nil
}

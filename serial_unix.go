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
	"regexp"
	"strings"
	"syscall"
	"time"

	"go.uber.org/multierr"
	"golang.org/x/sys/unix"

	"github.com/albenik/go-serial/v2/unixutils"
)

const FIONREAD = 0x541B

var zeroByte = []byte{0}

type port struct {
	handle int

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
	if err = unix.IoctlSetInt(h, unix.TIOCEXCL, 0); err != nil {
		return nil, newOSError(multierr.Append(err, unix.Close(h)))
	}

	p := newWithDefaults(name, &port{
		handle:           h,
		firstByteTimeout: true,
		readTimeout:      0,
		writeTimeout:     0,
	})

	// Setup serial port
	if err := p.Reconfigure(opts...); err != nil {
		return nil, p.closeAndReturnError(InvalidSerialPort, err)
	}

	if err = unix.SetNonblock(h, false); err != nil {
		return nil, p.closeAndReturnError(OsError, err)
	}

	fds := []int{0, 0}
	if err := syscall.Pipe(fds); err != nil {
		p.Close()
		return nil, p.closeAndReturnError(OsError, err)
	}
	p.internal.closePipeR = fds[0]
	p.internal.closePipeW = fds[1]

	return p, nil
}

func (p *Port) Reconfigure(opts ...Option) error {
	if err := p.checkValid(); err != nil {
		return err
	}

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
	_, err := unix.Write(p.internal.closePipeW, zeroByte)
	err = multierr.Combine(
		err,
		unix.Close(p.internal.closePipeW),
		unix.Close(p.internal.closePipeR),
		unix.IoctlSetInt(p.internal.handle, unix.TIOCNXCL, 0),
		unix.Close(p.internal.handle),
	)

	if err != nil {
		return newOSError(err)
	}
	return nil
}

func (p *Port) ReadyToRead() (uint32, error) {
	if err := p.checkValid(); err != nil {
		return 0, err
	}

	n, err := unix.IoctlGetInt(p.internal.handle, FIONREAD)
	if err != nil {
		return 0, newOSError(err)
	}
	return uint32(n), nil
}

func (p *Port) Read(b []byte) (int, error) {
	if err := p.checkValid(); err != nil {
		return 0, err
	}

	size, read := len(b), 0
	fds := unixutils.NewFDSet(p.internal.handle, p.internal.closePipeR)
	buf := make([]byte, size)

	now := time.Now()
	deadline := now.Add(time.Duration(p.internal.readTimeout) * time.Millisecond)

	for read < size {
		res, err := unixutils.Select(fds, nil, fds, deadline.Sub(now))
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return read, newOSError(err)
		}

		if res.IsReadable(p.internal.closePipeR) {
			return read, &PortError{code: PortClosed}
		}
		if !res.IsReadable(p.internal.handle) {
			return read, nil
		}

		n, err := unix.Read(p.internal.handle, buf[read:])
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return read, newOSError(err)
		}

		// read should always return some data as select reported, it was ready to read when we got to this point.
		if n == 0 {
			return read, &PortError{code: ReadFailed}
		}

		copy(b[read:], buf[read:read+n])
		read += n

		now = time.Now()
		if !now.Before(deadline) || p.internal.firstByteTimeout {
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
	fds := unixutils.NewFDSet(p.internal.handle)
	clFds := unixutils.NewFDSet(p.internal.closePipeR)

	deadline := time.Now().Add(time.Duration(p.internal.writeTimeout) * time.Millisecond)

	for written < size {
		n, err := unix.Write(p.internal.handle, b[written:])
		if err != nil {
			return written, newOSError(err)
		}

		if p.internal.writeTimeout == 0 {
			return n, nil
		}

		written += n
		now := time.Now()
		if p.internal.writeTimeout > 0 && !now.Before(deadline) {
			return written, nil
		}

		res, err := unixutils.Select(clFds, fds, fds, deadline.Sub(now))
		if err != nil {
			return written, newOSError(err)
		}

		if res.IsReadable(p.internal.closePipeR) {
			return written, &PortError{code: PortClosed}
		}

		if !res.IsWritable(p.internal.handle) {
			return written, &PortError{code: WriteFailed}
		}
	}
	return written, nil
}

func (p *Port) ResetInputBuffer() error {
	if err := p.checkValid(); err != nil {
		return err
	}

	if err := unix.IoctlSetInt(p.internal.handle, ioctlTcflsh, unix.TCIFLUSH); err != nil {
		return newOSError(err)
	}
	return nil
}

func (p *Port) ResetOutputBuffer() error {
	if err := p.checkValid(); err != nil {
		return err
	}

	if err := unix.IoctlSetInt(p.internal.handle, ioctlTcflsh, unix.TCOFLUSH); err != nil {
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

// TODO Second argument was forget here while interface type that forces to implement it was removed.
//      To support backward compatibility keep it here until version v3
func (p *Port) SetReadTimeoutEx(t, _ uint32) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	s, err := p.retrieveTermSettings()
	if err != nil {
		return err // port.retrieveTermSettings() already returned PortError
	}

	vtime := t / 100 // VTIME tenths of a second elapses between bytes
	if vtime > 255 || vtime*100 != t {
		return &PortError{code: InvalidTimeoutValue}
	}
	if vtime > 0 {
		s.termios.Cc[unix.VMIN] = 1
		s.termios.Cc[unix.VTIME] = uint8(t)
	} else {
		s.termios.Cc[unix.VMIN] = 0
		s.termios.Cc[unix.VTIME] = 0
	}

	if err = p.applyTermSettings(s); err != nil {
		return err // port.applyTermSettings() already returned PortError
	}

	p.internal.firstByteTimeout = false
	p.internal.readTimeout = int(t)
	return nil
}

func (p *Port) SetFirstByteReadTimeout(t uint32) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	if t > 0 && t < 0xFFFFFFFF {
		p.internal.firstByteTimeout = true
		p.internal.readTimeout = int(t)
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
		unix.IoctlSetInt(p.internal.handle, unix.TIOCNXCL, 0),
		unix.Close(p.internal.handle),
	)}
}

func (p *Port) setReadTimeoutValues(t int) {
	p.internal.firstByteTimeout = false
	p.internal.readTimeout = t
}

func (p *Port) setWriteTimeoutValues(t int) {
	p.internal.writeTimeout = t
}

func (p *Port) retrieveModemBitsStatus() (int, error) {
	s, err := unix.IoctlGetInt(p.internal.handle, unix.TIOCMGET)
	if err != nil {
		return 0, newOSError(err)
	}
	return s, nil
}

func (p *Port) applyModemBitsStatus(status int) error {
	if err := unix.IoctlSetInt(p.internal.handle, unix.TIOCMSET, status); err != nil {
		return newOSError(err)
	}
	return nil
}

func (p *Port) reconfigure() error {
	if err := p.checkValid(); err != nil {
		return err
	}

	s, err := p.retrieveTermSettings()
	if err != nil {
		return err // port.retrieveTermSettings() already returned PortError
	}

	if err := s.setBaudrate(p.baudRate); err != nil {
		return err
	}
	if err := s.setParity(p.parity); err != nil {
		return err
	}
	if err := s.setDataBits(p.dataBits); err != nil {
		return err
	}
	if err := s.setStopBits(p.stopBits); err != nil {
		return err
	}
	s.setRawMode(p.hupcl)
	// Explicitly disable RTS/CTS flow control
	s.setCtsRts(false)

	return p.applyTermSettings(s) // already returned PortError
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

func isHandleValid(h int) bool {
	return h != 0
}

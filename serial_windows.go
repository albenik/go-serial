//
// Copyright 2014-2018 Cristian Maglie.
// Copyright 2019 Veniamin Albaev <albenik@gmail.com>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package serial

// Useful links:
// https://msdn.microsoft.com/en-us/library/ff802693.aspx
// https://msdn.microsoft.com/en-us/library/ms810467.aspx
// https://github.com/pyserial/pyserial
// https://pythonhosted.org/pyserial/
// https://playground.arduino.cc/Interfacing/CPPWindows
// https://www.tldp.org/HOWTO/Serial-HOWTO-19.html

import "syscall"

var parityMap = map[Parity]byte{
	NoParity:    0,
	OddParity:   1,
	EvenParity:  2,
	MarkParity:  3,
	SpaceParity: 4,
}

var stopBitsMap = map[StopBits]byte{
	OneStopBit:           0,
	OnePointFiveStopBits: 1,
	TwoStopBits:          2,
}

type port struct {
	handle   syscall.Handle
	timeouts *commTimeouts
}

func Open(name string, opts ...Option) (*Port, error) {
	path, err := syscall.UTF16PtrFromString("\\\\.\\" + name)
	if err != nil {
		return nil, err
	}

	handle, err := syscall.CreateFile(
		path,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,   // exclusive access
		nil, // no security
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL|syscall.FILE_FLAG_OVERLAPPED,
		0)
	if err != nil {
		switch err {
		case syscall.ERROR_ACCESS_DENIED:
			return nil, &PortError{code: PortBusy}
		case syscall.ERROR_FILE_NOT_FOUND:
			return nil, &PortError{code: PortNotFound}
		}
		return nil, err
	}

	port := newWithDefaults(name, &port{
		handle: handle,
		timeouts: &commTimeouts{
			// Read blocks until done.
			ReadIntervalTimeout:        0,
			ReadTotalTimeoutMultiplier: 0,
			ReadTotalTimeoutConstant:   0,
			// Write blocks until done.
			WriteTotalTimeoutMultiplier: 0,
			WriteTotalTimeoutConstant:   0,
		},
	})
	if err = port.Reconfigure(opts...); err != nil {
		port.Close()
		return nil, err
	}

	return port, nil
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
	if err := p.checkValid(); err != nil {
		return err
	}

	if p.internal.handle == syscall.InvalidHandle {
		return nil
	}
	err := syscall.CloseHandle(p.internal.handle)
	p.internal.handle = syscall.InvalidHandle
	if err != nil {
		return &PortError{code: OsError, causedBy: err}
	}
	return nil
}

func (p *Port) ReadyToRead() (uint32, error) {
	if err := p.checkValid(); err != nil {
		return 0, err
	}

	var errs uint32
	var stat comstat
	if err := clearCommError(p.internal.handle, &errs, &stat); err != nil {
		return 0, &PortError{code: OsError, causedBy: err}
	}
	return stat.inque, nil
}

func (p *Port) Read(b []byte) (int, error) {
	if err := p.checkValid(); err != nil {
		return 0, err
	}

	if p.internal.handle == syscall.InvalidHandle {
		return 0, &PortError{code: PortClosed, causedBy: nil}
	}
	handle := p.internal.handle

	errs := new(uint32)
	stat := new(comstat)
	if err := clearCommError(handle, errs, stat); err != nil {
		return 0, &PortError{code: InvalidSerialPort, causedBy: err}
	}

	size := uint32(len(b))
	var readSize uint32
	if p.internal.timeouts.ReadTotalTimeoutConstant == 0 && p.internal.timeouts.ReadTotalTimeoutMultiplier == 0 {
		if stat.inque < size {
			readSize = stat.inque
		} else {
			readSize = size
		}
	} else {
		readSize = size
	}

	if readSize > 0 {
		var read uint32
		overlapped, err := createOverlappedStruct()
		if err != nil {
			return 0, &PortError{code: OsError, causedBy: err}
		}
		defer syscall.CloseHandle(overlapped.HEvent)
		err = syscall.ReadFile(handle, b[:readSize], &read, overlapped)
		if err != nil && err != syscall.ERROR_IO_PENDING {
			return 0, &PortError{code: OsError, causedBy: err}
		}
		err = getOverlappedResult(handle, overlapped, &read, true)
		if err != nil && err != syscall.ERROR_OPERATION_ABORTED {
			return 0, &PortError{code: OsError, causedBy: err}
		}
		return int(read), nil
	} else {
		return 0, nil
	}
}

func (p *Port) Write(b []byte) (int, error) {
	if err := p.checkValid(); err != nil {
		return 0, err
	}

	h := p.internal.handle
	errs := new(uint32)
	stat := new(comstat)
	if err := clearCommError(h, errs, stat); err != nil {
		return 0, &PortError{code: InvalidSerialPort, causedBy: err}
	}

	overlapped, err := createOverlappedStruct()
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(overlapped.HEvent)
	var written uint32
	err = syscall.WriteFile(h, b, &written, overlapped)
	if err == nil || err == syscall.ERROR_IO_PENDING || err == syscall.ERROR_OPERATION_ABORTED {
		err = getOverlappedResult(h, overlapped, &written, true)
		if err == nil || err == syscall.ERROR_OPERATION_ABORTED {
			return int(written), nil
		}
	}
	return int(written), err
}

func (p *Port) ResetInputBuffer() error {
	if err := p.checkValid(); err != nil {
		return err
	}

	return purgeComm(p.internal.handle, purgeRxClear|purgeRxAbort)
}

func (p *Port) ResetOutputBuffer() error {
	if err := p.checkValid(); err != nil {
		return err
	}

	return purgeComm(p.internal.handle, purgeTxClear|purgeTxAbort)
}

func (p *Port) SetDTR(dtr bool) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	// Like for RTS there are problems with the escapeCommFunction
	// observed behaviour was that DTR is set from false -> true
	// when setting RTS from true -> false
	// 1) Connect 		-> RTS = true 	(low) 	DTR = true 	(low) 	OKAY
	// 2) SetDTR(false) -> RTS = true 	(low) 	DTR = false (heigh)	OKAY
	// 3) SetRTS(false)	-> RTS = false 	(heigh)	DTR = true 	(low) 	ERROR: DTR toggled
	//
	// In addition this way the CommState Flags are not updated
	/*
		var res bool
		if dtr {
			res = escapeCommFunction(port.handle, commFunctionSetDTR)
		} else {
			res = escapeCommFunction(port.handle, commFunctionClrDTR)
		}
		if !res {
			return &PortError{}
		}
		return nil
	*/

	// The following seems a more reliable way to do it

	p.hupcl = dtr

	params := &dcb{}
	if err := getCommState(p.internal.handle, params); err != nil {
		return &PortError{causedBy: err}
	}

	params.Flags &= dcbDTRControlDisableMask
	if dtr {
		params.Flags |= dcbDTRControlEnable
	}

	if err := setCommState(p.internal.handle, params); err != nil {
		return &PortError{causedBy: err}
	}

	return nil
}

func (p *Port) SetRTS(rts bool) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	// It seems that there is a bug in the Windows VCP driver:
	// it doesn't send USB control message when the RTS bit is
	// changed, so the following code not always works with
	// USB-to-serial adapters.
	//
	// In addition this way the CommState Flags are not updated

	/*
		var res bool
		if rts {
			res = escapeCommFunction(port.handle, commFunctionSetRTS)
		} else {
			res = escapeCommFunction(port.handle, commFunctionClrRTS)
		}
		if !res {
			return &PortError{}
		}
		return nil
	*/

	// The following seems a more reliable way to do it

	params := &dcb{}
	if err := getCommState(p.internal.handle, params); err != nil {
		return &PortError{causedBy: err}
	}
	params.Flags &= dcbRTSControlDisableMask
	if rts {
		params.Flags |= dcbRTSControlEnable
	}
	if err := setCommState(p.internal.handle, params); err != nil {
		return &PortError{causedBy: err}
	}
	return nil
}

func (p *Port) GetModemStatusBits() (*ModemStatusBits, error) {
	if err := p.checkValid(); err != nil {
		return nil, err
	}

	var bits uint32
	if !getCommModemStatus(p.internal.handle, &bits) {
		return nil, &PortError{}
	}
	return &ModemStatusBits{
		CTS: (bits & msCTSOn) != 0,
		DCD: (bits & msRLSDOn) != 0,
		DSR: (bits & msDSROn) != 0,
		RI:  (bits & msRingOn) != 0,
	}, nil
}

func (p *Port) SetReadTimeout(t int) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	p.setReadTimeoutValues(t)
	return p.reconfigure()
}

func (p *Port) SetReadTimeoutEx(t, i uint32) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	p.internal.timeouts.ReadIntervalTimeout = i
	p.internal.timeouts.ReadTotalTimeoutMultiplier = 0
	p.internal.timeouts.ReadTotalTimeoutConstant = t
	return p.reconfigure()
}

func (p *Port) SetFirstByteReadTimeout(t uint32) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	if t > 0 && t < 0xFFFFFFFF {
		p.internal.timeouts.ReadIntervalTimeout = 0xFFFFFFFF
		p.internal.timeouts.ReadTotalTimeoutMultiplier = 0xFFFFFFFF
		p.internal.timeouts.ReadTotalTimeoutConstant = t
		return p.reconfigure()
	} else {
		return &PortError{code: InvalidTimeoutValue}
	}
}

func (p *Port) SetWriteTimeout(t int) error {
	if err := p.checkValid(); err != nil {
		return err
	}

	p.setWriteTimeoutValues(t)
	return p.reconfigure()
}

func (p *Port) setReadTimeoutValues(t int) {
	switch {
	case t < 0: // Block until the buffer is full.
		p.internal.timeouts.ReadIntervalTimeout = 0
		p.internal.timeouts.ReadTotalTimeoutMultiplier = 0
		p.internal.timeouts.ReadTotalTimeoutConstant = 0
	case t == 0: // Return immediately with or without data.
		p.internal.timeouts.ReadIntervalTimeout = 0xFFFFFFFF
		p.internal.timeouts.ReadTotalTimeoutMultiplier = 0
		p.internal.timeouts.ReadTotalTimeoutConstant = 0
	case t > 0: // Block until the buffer is full or timeout occurs.
		p.internal.timeouts.ReadIntervalTimeout = 0
		p.internal.timeouts.ReadTotalTimeoutMultiplier = 0
		p.internal.timeouts.ReadTotalTimeoutConstant = uint32(t)
	}
}

func (p *Port) setWriteTimeoutValues(t int) {
	switch {
	case t < 0:
		p.internal.timeouts.WriteTotalTimeoutMultiplier = 0
		p.internal.timeouts.WriteTotalTimeoutConstant = 0
	case t == 0:
		p.internal.timeouts.WriteTotalTimeoutMultiplier = 0
		p.internal.timeouts.WriteTotalTimeoutConstant = 0xFFFFFFFF
	case t > 0:
		p.internal.timeouts.WriteTotalTimeoutMultiplier = 0
		p.internal.timeouts.WriteTotalTimeoutConstant = uint32(t)
	}
}

func (p *Port) reconfigure() error {
	if err := setCommTimeouts(p.internal.handle, p.internal.timeouts); err != nil {
		p.Close()
		return &PortError{code: InvalidSerialPort, causedBy: err}
	}
	if err := setCommMask(p.internal.handle, evErr); err != nil {
		p.Close()
		return &PortError{code: InvalidSerialPort, causedBy: err}
	}
	params := &dcb{}
	if err := getCommState(p.internal.handle, params); err != nil {
		p.Close()
		return &PortError{code: InvalidSerialPort, causedBy: err}
	}
	params.Flags &= dcbRTSControlDisableMask
	params.Flags |= dcbRTSControlEnable
	params.Flags &= dcbDTRControlDisableMask
	if p.hupcl {
		params.Flags |= dcbDTRControlEnable
	}
	params.Flags &^= dcbOutXCTSFlow
	params.Flags &^= dcbOutXDSRFlow
	params.Flags &^= dcbDSRSensitivity
	params.Flags |= dcbTXContinueOnXOFF
	params.Flags &^= dcbInX
	params.Flags &^= dcbOutX
	params.Flags &^= dcbErrorChar
	params.Flags &^= dcbNull
	params.Flags &^= dcbAbortOnError
	params.XonLim = 2048
	params.XoffLim = 512
	params.XonChar = 17  // DC1
	params.XoffChar = 19 // C3

	params.BaudRate = uint32(p.baudRate)
	params.ByteSize = byte(p.dataBits)
	params.Parity = parityMap[p.parity]
	params.StopBits = stopBitsMap[p.stopBits]

	if err := setCommState(p.internal.handle, params); err != nil {
		p.Close()
		return &PortError{code: InvalidSerialPort, causedBy: err}
	}
	return nil
}

func GetPortsList() ([]string, error) {
	subKey, err := syscall.UTF16PtrFromString("HARDWARE\\DEVICEMAP\\SERIALCOMM\\")
	if err != nil {
		return nil, &PortError{code: ErrorEnumeratingPorts}
	}

	var h syscall.Handle
	if syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, subKey, 0, syscall.KEY_READ, &h) != nil {
		return nil, &PortError{code: ErrorEnumeratingPorts}
	}
	defer syscall.RegCloseKey(h)

	var valuesCount uint32
	if syscall.RegQueryInfoKey(h, nil, nil, nil, nil, nil, nil, &valuesCount, nil, nil, nil, nil) != nil {
		return nil, &PortError{code: ErrorEnumeratingPorts}
	}

	list := make([]string, valuesCount)
	for i := range list {
		var data [1024]uint16
		dataSize := uint32(len(data))
		var name [1024]uint16
		nameSize := uint32(len(name))
		if regEnumValue(h, uint32(i), &name[0], &nameSize, nil, nil, &data[0], &dataSize) != nil {
			return nil, &PortError{code: ErrorEnumeratingPorts}
		}
		list[i] = syscall.UTF16ToString(data[:])
	}
	return list, nil
}

func isHandleValid(h syscall.Handle) bool {
	return h != syscall.InvalidHandle
}

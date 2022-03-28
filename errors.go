package serial

// PortErrorCode is a code to easily identify the type of error.
type PortErrorCode int

const (
	PortErrorUnknown PortErrorCode = iota
	// PortBusy the serial port is already in used by another process.
	PortBusy
	// PortNotFound the requested port doesn't exist.
	PortNotFound
	// InvalidSerialPort the requested port is not a serial port.
	InvalidSerialPort
	// PermissionDenied the user doesn't have enough priviledges.
	PermissionDenied
	// InvalidSpeed the requested speed is not valid or not supported.
	InvalidSpeed
	// InvalidDataBits the number of data bits is not valid or not supported.
	InvalidDataBits
	// InvalidParity the selected parity is not valid or not supported.
	InvalidParity
	// InvalidStopBits the selected number of stop bits is not valid or not supported.
	InvalidStopBits
	// InvalidTimeoutValue Invalid timeout value passed.
	InvalidTimeoutValue
	// ErrorEnumeratingPorts an error occurred while listing serial port.
	ErrorEnumeratingPorts
	// PortClosed the port has been closed while the operation is in progress.
	PortClosed
	// FunctionNotImplemented the requested function is not implemented.
	FunctionNotImplemented
	// OsError Operating system function error.
	OsError
	// WriteFailed Port write failed.
	WriteFailed
	// ReadFailed Port read failed.
	ReadFailed
)

// PortError is a platform independent error type for serial ports.
type PortError struct {
	code    PortErrorCode
	wrapped error
}

// EncodedErrorString returns a string explaining the error code.
func (e PortError) EncodedErrorString() string {
	switch e.code {
	case PortErrorUnknown:
		return "error code not set"
	case PortBusy:
		return "serial port busy"
	case PortNotFound:
		return "serial port not found"
	case InvalidSerialPort:
		return "invalid serial port"
	case PermissionDenied:
		return "permission denied"
	case InvalidSpeed:
		return "port speed invalid or not supported"
	case InvalidDataBits:
		return "port data bits invalid or not supported"
	case InvalidParity:
		return "port parity invalid or not supported"
	case InvalidStopBits:
		return "port stop bits invalid or not supported"
	case InvalidTimeoutValue:
		return "timeout value invalid or not supported"
	case ErrorEnumeratingPorts:
		return "could not enumerate serial ports"
	case PortClosed:
		return "port has been closed"
	case FunctionNotImplemented:
		return "function not implemented"
	case OsError:
		return "operating system error"
	case ReadFailed:
		return "read filed"
	case WriteFailed:
		return "write failed"
	default:
		return "other error"
	}
}

// Error returns the complete error code with details on the cause of the error.
func (e PortError) Error() string {
	if e.wrapped != nil {
		return e.EncodedErrorString() + ": " + e.wrapped.Error()
	}
	return e.EncodedErrorString()
}

func (e PortError) Unwrap() error {
	return e.wrapped
}

// Code returns an identifier for the kind of error occurred.
func (e PortError) Code() PortErrorCode {
	return e.code
}

// Cause returns the cause for the error
// Deprecated: Use go1.13 error iterface Unwrap() instead.
func (e PortError) Cause() error {
	return e.Unwrap()
}

func newPortOSError(err error) *PortError {
	return &PortError{code: OsError, wrapped: err}
}

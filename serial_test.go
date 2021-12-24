package serial_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/albenik/go-serial/v2"
)

func TestPortNilReciever_Error(t *testing.T) {
	checkError := func(err error) {
		var portErr *serial.PortError
		require.ErrorAs(t, err, &portErr)
		assert.Equal(t, serial.PortClosed, portErr.Code())
		assert.ErrorIs(t, err, os.ErrInvalid)
	}

	t.Run("Close", func(t *testing.T) {
		checkError((*serial.Port)(nil).Close())
	})

	t.Run("Reconfigure", func(t *testing.T) {
		checkError((*serial.Port)(nil).Reconfigure())
	})

	t.Run("ReadyToRead", func(t *testing.T) {
		_, err := (*serial.Port)(nil).ReadyToRead()
		checkError(err)
	})

	t.Run("Read", func(t *testing.T) {
		_, err := (*serial.Port)(nil).Read(make([]byte, 16))
		checkError(err)
	})

	t.Run("Write", func(t *testing.T) {
		_, err := (*serial.Port)(nil).Write([]byte{1, 2, 3, 4, 5, 6, 7, 8})
		checkError(err)
	})

	t.Run("ResetInputBuffer", func(t *testing.T) {
		checkError((*serial.Port)(nil).ResetInputBuffer())
	})

	t.Run("ResetOutputBuffer", func(t *testing.T) {
		checkError((*serial.Port)(nil).ResetOutputBuffer())
	})

	t.Run("SetDTR", func(t *testing.T) {
		checkError((*serial.Port)(nil).SetDTR(false))
	})

	t.Run("SetRTS", func(t *testing.T) {
		checkError((*serial.Port)(nil).SetRTS(false))
	})

	t.Run("SetReadTimeout", func(t *testing.T) {
		checkError((*serial.Port)(nil).SetReadTimeout(1))
	})

	t.Run("SetReadTimeoutEx", func(t *testing.T) {
		checkError((*serial.Port)(nil).SetReadTimeoutEx(1, 0))
	})

	t.Run("SetFirstByteReadTimeout", func(t *testing.T) {
		checkError((*serial.Port)(nil).SetFirstByteReadTimeout(1))
	})

	t.Run("SetWriteTimeout", func(t *testing.T) {
		checkError((*serial.Port)(nil).SetWriteTimeout(1))
	})

	t.Run("GetModemStatusBits", func(t *testing.T) {
		_, err := (*serial.Port)(nil).GetModemStatusBits()
		checkError(err)
	})
}

func TestPortTestPortNilReciever_String(t *testing.T) {
	var p *serial.Port
	assert.Equal(t, "Error: <nil> port instance", p.String())
}

/*
func TestSerial_Operations(t *testing.T) {
	ports, err := serial.GetPortsList()
	require.NoError(t, err)

	if len(ports) == 0 {
		t.Log("No serial ports found")
	}

	for _, name := range ports {
		t.Logf("Found port: %q", name)
		port, err := serial.Open(name,
			serial.WithBaudrate(9600),
			serial.WithDataBits(8),
			serial.WithParity(serial.NoParity),
			serial.WithStopBits(serial.OneStopBit),
			serial.WithReadTimeout(1000),
			serial.WithWriteTimeout(1000),
			serial.WithHUPCL(false),
		)
		if err == nil {
			t.Logf("Port %q opened", name)
			if assert.NoError(t, port.Close()) {
				t.Logf("Port %q closed without errors", name)
			}
			continue
		}

		var portErr serial.PortError
		expected := errors.As(err, &portErr) && !in(portErr.Code(), serial.OsError, serial.InvalidSerialPort)
		if !expected {
			assert.NoError(t, err)
		}
	}
}

func in(code serial.PortErrorCode, list ...serial.PortErrorCode) bool {
	for _, c := range list {
		if code == c {
			return true
		}
	}
	return false
}
*/

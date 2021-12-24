package serial_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/albenik/go-serial/v2"
)

func TestPort_Nil_SetDTR(t *testing.T) {
	var p *serial.Port
	err := p.SetDTR(false)
	if assert.Error(t, err) && assert.IsType(t, new(serial.PortError), err) {
		portErr := err.(*serial.PortError)
		assert.Equal(t, serial.PortClosed, portErr.Code())
		assert.ErrorIs(t, os.ErrInvalid, portErr)
	}
}

func TestPort_Nil_String(t *testing.T) {
	var p *serial.Port
	assert.Equal(t, "Error: <nil> port instance", p.String())
}

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

package serial_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/albenik/go-serial/v2"

	"os"
)

func TestPort_Nil_SetDTR(t *testing.T) {
	var p *serial.Port
	err := p.SetDTR(false)
	if assert.Error(t, err) && assert.IsType(t, new(serial.PortError), err) {
		portErr := err.(*serial.PortError)
		assert.Equal(t, serial.PortClosed, portErr.Code())
		assert.Equal(t, os.ErrInvalid, portErr.Cause())
	}
}

func TestPort_Nil_String(t *testing.T) {
	var p *serial.Port
	assert.Equal(t, "Error: <nil> port instance", p.String())
}

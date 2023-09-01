//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package serial_test

import (
	"fmt"
	"log"

	"github.com/bclswl0827/go-serial/v2"
)

func ExamplePort_Reconfigure() {
	port, err := serial.Open("/dev/ttyACM0")
	if err != nil {
		log.Fatal(err)
	}
	if err := port.Reconfigure(
		serial.WithBaudrate(9600),
		serial.WithDataBits(8),
		serial.WithParity(serial.NoParity),
		serial.WithStopBits(serial.OneStopBit),
		serial.WithReadTimeout(1000),
		serial.WithWriteTimeout(1000),
	); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Port set to 9600 N81")
}

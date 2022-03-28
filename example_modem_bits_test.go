//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package serial_test

import (
	"fmt"
	"log"
	"time"

	"github.com/albenik/go-serial/v2"
)

func ExamplePort_GetModemStatusBits() {
	// Open the first serial port detected at 9600bps N81
	port, err := serial.Open("/dev/ttyACM1",
		serial.WithBaudrate(9600),
		serial.WithDataBits(8),
		serial.WithParity(serial.NoParity),
		serial.WithStopBits(serial.OneStopBit),
		serial.WithReadTimeout(1000),
		serial.WithWriteTimeout(1000),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer port.Close()

	count := 0
	for count < 25 {
		status, err := port.GetModemStatusBits()
		if err != nil {
			log.Println(err) // DO NOT USER log.Fatal or `port.Close()` deferred call will never happened!
			return
		}
		fmt.Printf("Status: %+v\n", status)

		time.Sleep(time.Second)
		count++
		if count == 5 {
			if err = port.SetDTR(false); err != nil {
				log.Println(err)
				return
			}
			fmt.Println("Set DTR OFF")
		}
		if count == 10 {
			if err = port.SetDTR(true); err != nil {
				log.Println(err)
				return
			}
			fmt.Println("Set DTR ON")
		}
		if count == 15 {
			if err = port.SetRTS(false); err != nil {
				log.Println(err)
				return
			}
			fmt.Println("Set RTS OFF")
		}
		if count == 20 {
			if err = port.SetRTS(true); err != nil {
				log.Println(err)
				return
			}
			fmt.Println("Set RTS ON")
		}
	}
}

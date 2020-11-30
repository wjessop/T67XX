package main

import (
	"log"
	"os"

	"github.com/wjessop/t67xx"
	"golang.org/x/exp/io/i2c"
)

const (
	// We can change the sensor address on the bus if we want to, but it defaults
	// to 0x21
	t67XXSensorAddress    = 0x21
	t67XXSensorNewAddress = 0x22
)

func main() {
	// Open an i2c bus that we can pass to the driver
	device, err := i2c.Open(&i2c.Devfs{Dev: "/dev/i2c-1"}, t67XXSensorAddress)
	if err != nil {
		log.Fatalf("Couldn't open the T67XX sensor at %x, error was %v", t67XXSensorAddress, err)
	}

	// Create the driver
	driver := &t67xx.T67XX{}
	driver.Device = device

	// For now the library needs a logger to be provided. It needs to satisfy the
	// following interface:
	//
	// type Logger interface {
	// 	Debug(...interface{})
	// 	Debugf(string, ...interface{})
	// 	Fatalf(string, ...interface{})
	// }
	log := log.New(os.Stderr, "T67XX", log.LstdFlags)
	driver.SetLogger(log)

	if err := driver.SetAddress(byte(t67XXSensorNewAddress)); err != nil {
		log.Fatal(err)
	}
}

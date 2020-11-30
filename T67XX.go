package t67xx

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/io/i2c"
)

const (
	// How long to sleep after sending a command before the sensor response should
	// be read. From the Datasheet:
	//
	//   "It is suggested that the master send the request, wait 5 to 10 milliseconds
	//    and then ask for the response. This time does depend on bus loading and
	//    board layout but carefully constructed test setups have dem-onstrated that
	//    the sensor can respond within 1 millisecond in controlled conditions with
	//    a data rate of 100kbps. The suggested delay of 10 milliseconds should be
	//    adequate for almost all conceivable cases
	commandSleep = 10 * time.Millisecond
)

var (
	statusBitData = []BitValue{
		{0x1, "Error condition"},
		{0x2, "Flash error"},
		{0x4, "Calibration error"},
		{0x100, "RS-232"},
		{0x200, "RS-485"},
		{0x400, "I2C"},
		{0x800, "Warm-up mode"},
		{0x8000, "Single point calibration"},
	}
)

// Logger is the definition of the logger interface we need
type Logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
	Fatalf(string, ...interface{})
}

// T67XX encapsulates communications with the T67XX CO₂ sensor
type T67XX struct {
	Device *i2c.Device
	log    Logger
}

// SetLogger sets the logger to use
func (t *T67XX) SetLogger(l Logger) {
	t.log = l
}

// FirmwareVersion returns the a sensors firmware
func (t *T67XX) FirmwareVersion() (int, error) {
	// Write the command
	if err := t.Device.Write([]byte{0x04, 0x13, 0x89, 0x00, 0x01}); err != nil {
		return 0, err
	}

	time.Sleep(commandSleep)

	// Read the sensor data
	b := make([]byte, 4)
	if err := t.Device.Read(b); err != nil {
		return 0, err
	}

	t.log.Debugf("Read firmware version bytes: %v", b)
	t.log.Debugf("Raw firmware version bytes: % 08b", b)

	return 1, nil
}

// GasPPM returns the CO₂ parts per million measured on the sensor
func (t *T67XX) GasPPM() (int, error) {
	// Write the command
	if err := t.Device.Write([]byte{0x04, 0x13, 0x8b, 0x00, 0x01}); err != nil {
		return 0, err
	}

	time.Sleep(10 * time.Millisecond)

	// Read the sensor data
	b := make([]byte, 4)
	if err := t.Device.Read(b); err != nil {
		return 0, err
	}

	return int(b[2])*256 + int(b[3]), nil
}

// PrintStatus prints the status of the sensor
func (t *T67XX) PrintStatus() error {
	// Write the command
	if err := t.Device.Write([]byte{0x04, 0x13, 0x8a, 0x00}); err != nil {
		return err
	}

	time.Sleep(commandSleep)

	// Read the sensor data
	b := make([]byte, 2)
	if err := t.Device.Read(b); err != nil {
		return err
	}

	t.log.Debugf("Read status bytes: %v\n", b)
	t.log.Debugf("Raw status bytes: % 08b\n", b)
	fmt.Printf("Status bits set: %s", strings.Join(Bitmask(binary.BigEndian.Uint16(b)).ListDescriptions(statusBitData), ", "))
	return nil
}

// Reset resets the sensor. You will need to make sure the sensor is available
// before getting a new reading
func (t *T67XX) Reset() error {
	if err := t.Device.Write([]byte{0x05, 0x03, 0xe8, 0xff, 0x00}); err != nil {
		return err
	}

	return nil
}

// EnableABC enables the ABC calibration. From the datasheet:
//
//   "ABC LOGIC™ Automatic Background Logic, or ABC Logic™, is a patented
//    self-calibration technique that is designed to be used in applications where
//    concentrations will drop to outside ambient conditions (400 ppm) at least
//    three times in a 7 days, typically during unoccupied periods. Full accuracy
//    to be achieved utilizing ABC Logic™. With ABC Logic™ enabled, the sensor will
//    typically reach its operational accuracy after 24 hours of continuous
//    operation at a condition that it was exposed to ambient reference levels of
//    air at 400 ppm CO2. Sensor will maintain accuracy specifications with ABC
//    Logic™ enabled, given that it is at least four times in 21 days exposed to
//    the reference value and this reference value is the lowest concentration
//    to which the sensor is exposed. ABC Logic™ requires continuous operation of
//    the sensor for periods of at least 24 hours.
//
//    Note: Applies when used in typical residential ambient air. Consult Telaire
//    if other gases or corrosive agents are part of the application environment."
func (t *T67XX) EnableABC() error {
	// Write the command
	if err := t.Device.Write([]byte{0x05, 0x03, 0xee, 0xff, 0x00}); err != nil {
		return err
	}

	return nil
}

// def calibrate(self):
// buffer = array.array('B', [0x05, 0x03, 0xec, 0xff, 0x00])
// self.dev.write(buffer)
// time.sleep(0.1)
// data = self.dev.read(5)
// buffer = array.array('B', data)
// return buffer[3]*256+buffer[3]

// SetAddress sets the i2c address of the sensor
func (t *T67XX) SetAddress(address byte) error {
	if address < 0x03 || address > 0x77 {
		t.log.Fatalf("Address should be in the range 0x03 -> 0x77, you requested address 0x%x", int(address))
	}

	if err := t.Device.Write([]byte{0x06, 0x0f, 0xa5, 0x00, address}); err != nil {
		return err
	}

	time.Sleep(time.Second)

	if err := t.Reset(); err != nil {
		return err
	}

	time.Sleep(time.Second)

	return nil
}

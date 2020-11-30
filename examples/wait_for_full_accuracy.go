package main

import (
	"log"
	"os"
	"time"

	"github.com/wjessop/t67xx"
	"golang.org/x/exp/io/i2c"
)

const (
	// We can change the sensor address on the bus if we want to, but it defaults
	// to 0x21
	t67XXSensorAddress = 0x21
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
	if err := driver.EnableABC(); err != nil {
		log.Fatal("Could not enable ABC calibration on the sensor", err)
	}

	// Create a signal channel that will be closed when the sensor reaches full accuracy
	accuracyChan := make(chan interface{})

	go func(driver *t67xx.T67XX) {
		// Sleep in the background until the sensor has been powered up long enough
		// to achieve full accuracy.
		err := driver.SleepUntilFullAccuracy()
		if err != nil {
			log.Fatal("Error sleeping until full accuracy", err)
		}

		// Close the signal channel then exit the goroutine as we no-longer need it.
		close(accuracyChan)
	}(driver)

	// Now we can read the CO₂ readings in a loop, taking care to discard any
	// spurious readings.
	for {
		select {
		case <-accuracyChan:
			// A successful read on the closed channel indicates that the sensor is
			// now fully accurate.
			co2Reading, err := driver.GasPPM()
			if err != nil {
				log.Fatal(err)
			}

			// The sensors I have sometimes give spurious readings. Let's discount them.
			// Adjust these values based on the baseline CO₂ reading you expect. The max is
			// the measurement limit according to the datasheet, but i've seen values well
			// over 10,000.
			if co2Reading > 5000 || co2Reading < 200 {
				log.Printf("Reading of %d from CO₂ sensor was out of allowed bounds", co2Reading)
			} else {
				log.Printf("Got CO₂ reading of %d from CO₂ sensor", co2Reading)
			}
		default:
			log.Print("Skipping CO₂ reading as the sensor has not yet achieved full accuracy")
		}

		time.Sleep(10)
	}
}

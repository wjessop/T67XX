# T67XX CO₂ sensor control using Go

The T67XX module is a relatively cheap and easy to use CO₂ sensor that can be used (amongst other methods) over an [I²C bus](https://en.wikipedia.org/wiki/I%C2%B2C). You can find data on the sensor module on [the datasheet](docs/Manual-AMP-0002-T6713-Sensor.pdf)

This Go module interfaces with the sensor and allows you configure it, and to read values from it.

## Things to note

* The sensor reaches what it calls "operational accuracy" after 120 seconds, and "full accuracy" after 10 minutes.
* You probably want to enable ABC calibration on the device.

Both of these things are documented in the code and examples.

## Simple usage

```go
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
	//    air at 400 ppm CO₂. Sensor will maintain accuracy specifications with ABC
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

	// Now we can read the CO₂ readings in a loop, taking care to discard any
	// spurious readings.
	for {
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

		time.Sleep(10)
	}
}
```

## Waiting until the sensor reaches full accuracy

For my own convenience I added methods to allow for waiting until the sensor has reached full accuracy. They only work in Linux, because that's where I have my sensors attached. They don't *really* belong in this library, but they're here if you want them. If you object to their presence, or you're not using Linux then you can simply ignore them and do the book keeping yourself.

The methods rely on time since the machine was booted to determine wether the sensor has reached full accuracy yet, assuming that the sensors were plugged in at system boot. From the datasheet:

> The sensor is capable of responding to commands after power on, but operational accuracy of sensor won’t happen until 120 sec have elapsed. The sensor will reach full accuracy / warm up after 10 min. of operation.

### Using the SleepUntilFullAccuracy() function

The `SleepUntilFullAccuracy()` function does just that. Based on the time since last system boot it sleeps for the remaining 10 minute duration until the sensor should have reached full accuracy. To avoid blocking your entire program (you may be reading other sensors) you can use a goroutine and channel to signal the rest of the program.

First, create a channel to use as a signal, then in a goroutine close it when the `SleepUntilFullAccuracy()` function returns:

```go
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
```

Now, the rest of our program looks similar to the simple example but with a channel select which will only start to succeed if the signal channel is closed:

```go
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
```

### Using the SensorIsAtFullAccuracy() method

This method simply re-calculates the seconds since system boot each time it is called, and as such is not as efficient for long term/repeated use as the previous method:

```go
accurate, err := driver.SensorIsAtFullAccuracy()
if err != nil {
	log.Fatal("Error sleeping until full accuracy", err)
}

if accurate {
	// do reading
}
```

## Other functions

### Changing the address of the sensor on the I²C bus

I used this so I could have 4 sensors all on the same bus so I could check them against each other to check for deviation.

```go
t67XXSensorNewAddress := 0x22 // The sensor defaults to 0x21

if err := driver.SetAddress(byte(t67XXSensorNewAddress)); err != nil {
	log.Fatal(err)
}
```

### Printing the status

The sensor reports some data, mine have only ever reported having the "Calibration error", "I2C" and "Single point calibration" bits set though, so YMMV.

```go
if err := driver.PrintStatus(); err != nil {
	log.Fatal(err)
}
```

## TODO, and things I don't need so won't do but you can

* Meta methods to determine wether the sensor has reached "operational accuracy".
* Something useful with the status.
* Single point calibration triggering and reporting.
* Firmware. Right now it just returns 1.

package t67xx

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Only works on Linux currently
func (t *T67XX) secondsSinceSystemBoot() (int64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		t.log.Debug("Could not read /proc/stat: ", err)
		return 0, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var line string
	var seconds int64

	for scanner.Scan() {
		line = scanner.Text()
		if strings.HasPrefix(line, "btime") {
			seconds, err = strconv.ParseInt(line[6:], 10, 64)
			if err != nil {
				return 0, err
			}

			break
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return seconds, nil
}

// Only works on Linux currently
func (t *T67XX) secondsUntilFullAccuracy() (time.Duration, error) {
	secondsSinceSystemBoot, err := t.secondsSinceSystemBoot()
	if err != nil {
		return 0, errors.Wrap(err, "could not get seconds since syetem boot")
	}

	tenMinuesAfterSystemBoot := time.Unix(secondsSinceSystemBoot+600, 0)
	now := time.Now()
	diff := tenMinuesAfterSystemBoot.Sub(now)
	return diff, nil
}

// SensorIsAtFullAccuracy returns true if the sensor has reached it's full
// accuracy, or false otherwise.
//
// From the datasheet:
//
//   "The sensor is capable of responding to commands after power on, but operational
//    accuracy of sensor won’t happen until 120 sec have elapsed. The sensor will
//    reach full accuracy / warm up after 10 min. of operation."
//
// This code assumed that the sensor was plugged in when the system was booted
// and hasn't been removed since.
//
// Only works in Linux currently.
func (t *T67XX) SensorIsAtFullAccuracy() (bool, error) {
	secondsUntilFullAccuracy, err := t.secondsUntilFullAccuracy()
	if err != nil {
		return false, errors.Wrap(err, "could not determine the number of seconds until full sensor accuracy")
	}

	if secondsUntilFullAccuracy > 0 {
		t.log.Debug("System boot was more than 10 minutes ago, sensors should have reached full accuracy")
		return true, nil
	}

	t.log.Debugf("System boot was less than 10 minutes ago and will reach full accuracy in %d seconds", secondsUntilFullAccuracy)
	return false, nil
}

// SleepUntilFullAccuracy sleeps for 10 minutes from the time the system was booted,
// which is the amount of time it takes for the sensor to achieve full accuracy.
//
// If the system was booted more than 10 minutes ago it returns immetiately. Only
// works on Linux. Not sure if the file is available/works in containers.
//
// This code (and the associated functions it calls) probably aren't really too
// relevant to the sensor driver itself, they're more her for my convenience.
func (t *T67XX) SleepUntilFullAccuracy() error {
	secondsUntilFullAccuracy, err := t.secondsUntilFullAccuracy()
	if err != nil {
		return errors.Wrap(err, "could not determine the number of seconds until full sensor accuracy")
	}

	if secondsUntilFullAccuracy > 0 {
		t.log.Debugf("System boot was less than 10 minutes ago, sleeping for %d seconds until full sensor accuracy", secondsUntilFullAccuracy)
		<-time.After(secondsUntilFullAccuracy)
	}

	t.log.Debug("System boot was more than 10 minutes ago, sensors should have reached full accuracy")
	return nil
}

package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/warthog618/gpio"
)

const (
	defaultAnemRadius = 9
	defaultInterval   = 5
	defaultAnemFactor = 1.18
	defaultPin        = gpio.GPIO5
)

func main() {
	pinNumber := flag.Int("pin", defaultPin, "GPIO pin on which to listen")
	anemRadius := flag.Float64("radius", defaultAnemRadius, "Radius of the anemometer")
	interval := flag.Int("interval", defaultInterval, "Time interval between readings")
	anemFactor := flag.Float64("anem-factor", defaultAnemFactor, "The calibration anemometer factor")

	anemCircum := (2 * math.Pi) * (*anemRadius)
	intervalDuration := time.Duration(*interval) * time.Second

	fmt.Printf("Running with radius=%f, interval=%d, anemometer factor=%f, pin=%d\n", *anemRadius, *interval,
		*anemFactor, *pinNumber)

	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()

	pin := gpio.NewPin(*pinNumber)
	pin.Input()
	pin.PullUp()
	defer pin.Unwatch()

	pinHighs := 0
	pinLock := new(sync.Mutex)

	resetHighs := func() {
		pinLock.Lock()
		pinHighs = 0
		pinLock.Unlock()
	}

	err = pin.Watch(gpio.EdgeRising, func(pin *gpio.Pin) {
		pinLock.Lock()
		if pin.Read() {
			pinHighs++
		}
		pinLock.Unlock()
	})
	if err != nil {
		panic(err)
	}

	stopTimerSig := make(chan os.Signal, 1)
	signal.Notify(stopTimerSig, os.Interrupt)
	defer signal.Stop(stopTimerSig)

	for {
		select {
		case <-stopTimerSig:
			return
		case <-time.After(intervalDuration):
			pinLock.Lock()
			highs := pinHighs
			pinLock.Unlock()
			rotations := float64(highs) / 2 // anemometer triggers twice per rotation
			distance := (anemCircum * rotations) / 100000
			speed := (distance / float64(*interval)) * 3600 * (*anemFactor)

			fmt.Printf("Speed = %fkm/h\n", speed)
			resetHighs()
		}
	}
}

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/warthog618/gpio"
)

const (
	defaultInterval = 5
	defaultPin      = gpio.GPIO6
	rainPerTip      = 0.2794
)

func main() {
	pinNumber := flag.Int("pin", defaultPin, "GPIO pin on which to listen")
	interval := flag.Int("interval", defaultInterval, "Time interval between readings")

	flag.Parse()

	intervalDuration := time.Duration(*interval) * time.Second

	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer func() {
		err = gpio.Close()
		fmt.Printf("failed to close gpio: %v\n", err)
	}()

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
			rainfall := float64(highs) * rainPerTip

			fmt.Printf("Rainfall = %.2fmm over %d seconds\n", rainfall, *interval)
			resetHighs()
		}
	}
}

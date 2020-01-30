package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/warthog618/gpio/spi/mcp3w0c"

	"github.com/warthog618/gpio"
)

const (
	defaultClock    = 500 * time.Nanosecond
	defaultClockPin = gpio.GPIO11
	defaultDOutPin  = gpio.GPIO9
	defaultDInPin   = gpio.GPIO10
	defaultCSPin    = gpio.GPIO8
	defaultChannel  = 0
)

var voltsToDegrees = map[string]float32{
	"0.4": 0.0,
	"1.4": 22.5,
	"1.2": 45.0,
	"2.8": 67.5,
	"2.7": 90.0,
	"2.9": 112.5,
	"2.2": 135.0,
	"2.5": 157.5,
	"1.8": 180.0,
	"2.0": 202.5,
	"0.7": 225.0,
	"0.8": 247.5,
	"0.1": 270.0,
	"0.3": 292.5,
	"0.2": 315.0,
	"0.6": 337.5,
}

func main() {
	clock := flag.Duration("clock", defaultClock, "Time between cycle edges")
	clockPinNumber := flag.Int("clk", defaultClockPin, "Pin on which to listen for clk")
	dOutPinNumber := flag.Int("dout", defaultDOutPin, "Pin on which to listen for dout")
	dInPinNumber := flag.Int("din", defaultDInPin, "Pin on which to listen for din")
	csPinNumber := flag.Int("cs", defaultCSPin, "Pin on which to listen for cs")
	channel := flag.Int("channel", defaultChannel, "Channel to read on")
	interval := flag.Duration("interval", 1*time.Second, "Time interval between readings")

	flag.Parse()

	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer func() {
		err = gpio.Close()
		if err != nil {
			fmt.Printf("failed to close gpio: %v\n", err)
		}
	}()

	adc := mcp3w0c.NewMCP3008(
		*clock,
		*clockPinNumber,
		*csPinNumber,
		*dInPinNumber,
		*dOutPinNumber,
	)
	defer adc.Close()

	stopTimerSig := make(chan os.Signal, 1)
	signal.Notify(stopTimerSig, os.Interrupt)
	defer signal.Stop(stopTimerSig)

	for {
		select {
		case <-stopTimerSig:
			return
		case <-time.After(*interval):
			reading := adc.Read(*channel)
			voltage := float64(reading) / 1023 * 3.3
			voltageString := fmt.Sprintf("%.1f", voltage)
			degrees, ok := voltsToDegrees[voltageString]
			if !ok {
				fmt.Printf("Got a voltage that does not have a corresponding degrees:%s\n", voltageString)
				continue
			}

			fmt.Printf("%.1f\n", degrees)
		}
	}
}

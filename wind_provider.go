package weatherstn

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/warthog618/gpio"
	"github.com/warthog618/gpio/spi/mcp3w0c"
)

const (
	// SEN08942AnemFactor is the calibration factor for the anemometer.
	SEN08942AnemFactor = 1.18

	// SEN08942AnemCircum is the circumference of the anemometer.
	SEN08942AnemCircum = (2 * math.Pi) * 9.0

	// SEN08942Voltage is the voltage supplied to the wind vane.
	SEN08942Voltage = 3.3

	// SEN08942NumADCValues is the number of values that can be reported by the wind vane.
	SEN08942NumADCValues = 1023 // The value returned is 0-1023 where 1023 = 3.3.

	cmInKM     = 100000.0
	secsInHour = 3600
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

// WindSensorProvider provides a way to setup and collect wind data readings.
type WindSensorProvider interface {
	SensorProvider
	Readings() (*WindReadings, error)
}

// WindReadings are the sensor readings about measurements such as wind speed.
type WindReadings struct {
	Speed     float64 // km/h
	Direction float32 // degrees
	Gust      float64 // km/h
}

// SEN08942WindSensorProvider uses the SEN08942 weather kit to provide wind sensor readings.
type SEN08942WindSensorProvider struct {
	pin *gpio.Pin
	adc *mcp3w0c.MCP3w0c

	pinLock    sync.Mutex
	highCounts int

	totalSpeed float64
	speedsRead float64
	maxGust    float64
	speedsLock sync.Mutex

	anemPinNumber int
	anemInterval  time.Duration

	vaneClkPinNumber  int
	vaneCSPinNumber   int
	vaneDInPinNumber  int
	vaneDOutPinNumber int
	vaneChannel       int

	haltCh   chan struct{}
	haltedCh chan struct{}
}

// SEN08942WindSensorProviderConfig is used for setup of the SEN08942.
type SEN08942WindSensorProviderConfig struct {
	AnemPinNumber int
	AnemInterval  time.Duration

	VaneClkPinNumber  int
	VaneCSPinNumber   int
	VaneDInPinNumber  int
	VaneDOutPinNumber int
	VaneChannel       int
}

// NewSEN08942WindSensorProvider returns a new SEN08942WindSensorProvider.
func NewSEN08942WindSensorProvider(config SEN08942WindSensorProviderConfig) *SEN08942WindSensorProvider {
	return &SEN08942WindSensorProvider{
		anemPinNumber:     config.AnemPinNumber,
		anemInterval:      config.AnemInterval,
		vaneClkPinNumber:  config.VaneClkPinNumber,
		vaneCSPinNumber:   config.VaneCSPinNumber,
		vaneDInPinNumber:  config.VaneDInPinNumber,
		vaneDOutPinNumber: config.VaneDOutPinNumber,
		vaneChannel:       config.VaneChannel,

		haltCh:   make(chan struct{}),
		haltedCh: make(chan struct{}),
	}
}

// Connect sets up the connections to pins and creates watchers.
func (wr *SEN08942WindSensorProvider) Connect() error {
	err := gpio.Open()
	if err != nil && !errors.Is(err, gpio.ErrAlreadyOpen) {
		return err
	}

	pin := gpio.NewPin(wr.anemPinNumber)
	pin.Input()
	pin.PullUp()
	err = pin.Watch(gpio.EdgeRising, wr.onPinHigh)
	if err != nil {
		closeErr := gpio.Close()
		if closeErr != nil {
			log.WithError(closeErr).
				WithField("component", "wind provider").
				Error("gpio failed to close")
		}
		return err
	}

	wr.pin = pin

	adc := mcp3w0c.NewMCP3008(
		500*time.Nanosecond,
		wr.vaneClkPinNumber,
		wr.vaneCSPinNumber,
		wr.vaneDInPinNumber,
		wr.vaneDOutPinNumber,
	)

	wr.adc = adc
	go wr.monitorWindSpeed()

	return nil
}

// Disconnect closes the connections to the pins.
func (wr *SEN08942WindSensorProvider) Disconnect() {
	wr.haltCh <- struct{}{}
	<-wr.haltedCh

	wr.pinLock.Lock()
	wr.pin.Unwatch()
	wr.pinLock.Unlock()
	wr.adc.Close()
	err := gpio.Close()
	if err != nil && !errors.Is(err, unix.EINVAL) { // We're using gpio in 2 places and closing it twice causes EINVAL
		log.WithError(err).
			WithField("component", "wind provider").
			Error("gpio failed to close")
	}
}

// Readings returns the set of WindReadings provided by the SEN08942.
func (wr *SEN08942WindSensorProvider) Readings() (*WindReadings, error) {
	dirReading := wr.adc.Read(wr.vaneChannel)
	voltage := float64(dirReading) / SEN08942NumADCValues * SEN08942Voltage
	voltageString := fmt.Sprintf("%.1f", voltage)
	degrees, ok := voltsToDegrees[voltageString]
	if !ok {
		log.WithField("adc reading", dirReading).
			WithField("voltage", voltage).
			WithField("component", "wind provider").
			Error("unrecognised bearing for direction reading")

		degrees = -1 // 0 is a valid reading so don't use that.
	}

	wr.speedsLock.Lock()
	totalSpeed := wr.totalSpeed
	speedsSeen := wr.speedsRead
	gust := wr.maxGust
	wr.speedsLock.Unlock()
	wr.resetSpeeds()

	averageSpeed := 0.0
	if totalSpeed > 0 {
		averageSpeed = totalSpeed / speedsSeen
	}

	return &WindReadings{
		Speed:     averageSpeed,
		Direction: degrees,
		Gust:      gust,
	}, nil
}

func (wr *SEN08942WindSensorProvider) onPinHigh(pin *gpio.Pin) {
	if pin.Read() {
		wr.pinLock.Lock()
		wr.highCounts++
		wr.pinLock.Unlock()
	}
}

func (wr *SEN08942WindSensorProvider) resetPinHighs() {
	wr.pinLock.Lock()
	wr.highCounts = 0
	wr.pinLock.Unlock()
}

func (wr *SEN08942WindSensorProvider) resetSpeeds() {
	wr.speedsLock.Lock()
	wr.maxGust = 0
	wr.totalSpeed = 0
	wr.speedsRead = 0
	wr.speedsLock.Unlock()
}

func (wr *SEN08942WindSensorProvider) monitorWindSpeed() {
	for {
		wr.resetPinHighs()
		select {
		case <-wr.haltCh:
			wr.haltedCh <- struct{}{}
			return
		case <-time.After(wr.anemInterval):
		}

		wr.pinLock.Lock()
		highs := wr.highCounts
		wr.pinLock.Unlock()

		rotations := float64(highs) / 2 // Anemometer triggers twice per rotation
		distance := (SEN08942AnemCircum * rotations) / cmInKM
		speed := (distance / (float64(wr.anemInterval / time.Second))) * secsInHour * SEN08942AnemFactor

		wr.speedsLock.Lock()
		wr.totalSpeed += speed
		wr.speedsRead++
		if speed > wr.maxGust {
			wr.maxGust = speed
		}
		wr.speedsLock.Unlock()
	}
}

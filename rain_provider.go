package main

import (
	"errors"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	log "github.com/sirupsen/logrus"

	"github.com/warthog618/gpio"
)

const RainfallMMPerTip = 0.02794

// RainSensorProvider provides a way to setup and collect rain data readings.
type RainSensorProvider interface {
	SensorProvider
	Readings() (*RainReadings, error)
}

// RainReadings are the sensor readings about measurements such as rainfall.
type RainReadings struct {
	Rainfall float64 // mm
}

type SEN08942RainSensorProvider struct {
	pin *gpio.Pin

	pinNumber int
	interval  time.Duration

	totalRainfall float64
	rainfallLock  sync.Mutex

	pinLock    sync.Mutex
	highCounts int

	haltCh   chan struct{}
	haltedCh chan struct{}
}

// SEN08942SensorProviderConfig is used for setup of the SEN08942.
type SEN08942RainSensorProviderConfig struct {
	PinNumber int
	Interval  time.Duration
}

func NewSEN08942RainSensorProvider(config SEN08942RainSensorProviderConfig) *SEN08942RainSensorProvider {
	return &SEN08942RainSensorProvider{
		pinNumber: config.PinNumber,
		interval:  config.Interval,

		haltCh:   make(chan struct{}),
		haltedCh: make(chan struct{}),
	}
}

func (rsp *SEN08942RainSensorProvider) Connect() error {
	err := gpio.Open()
	if err != nil && !errors.Is(err, gpio.ErrAlreadyOpen) {
		return err
	}

	pin := gpio.NewPin(rsp.pinNumber)
	pin.Input()
	pin.PullUp()

	err = pin.Watch(gpio.EdgeRising, rsp.onPinHigh)
	if err != nil {
		closeErr := gpio.Close()
		if closeErr != nil {
			log.WithError(closeErr).
				WithField("component", "rain provider").
				Error("gpio failed to close")
		}
		return err
	}

	rsp.pin = pin

	go rsp.monitorRainfall()

	return nil
}

// Disconnect closes the connections to the pins.
func (rsp *SEN08942RainSensorProvider) Disconnect() {
	rsp.haltCh <- struct{}{}
	<-rsp.haltedCh

	rsp.pinLock.Lock()
	rsp.pin.Unwatch()
	rsp.pinLock.Unlock()

	err := gpio.Close()
	if err != nil && !errors.Is(err, unix.EINVAL) { // We're using gpio in 2 places and closing it twice causes EINVAL
		log.WithError(err).
			WithField("component", "rain provider").
			Error("gpio failed to close")
	}
}

// Readings returns the set of RainReadings provided by the SEN08942.
func (rsp *SEN08942RainSensorProvider) Readings() (*RainReadings, error) {
	rsp.rainfallLock.Lock()
	totalRainfall := rsp.totalRainfall
	rsp.rainfallLock.Unlock()
	rsp.resetRainfall()

	return &RainReadings{
		Rainfall: totalRainfall,
	}, nil
}

func (rsp *SEN08942RainSensorProvider) onPinHigh(pin *gpio.Pin) {
	if pin.Read() {
		rsp.pinLock.Lock()
		rsp.highCounts++
		rsp.pinLock.Unlock()
	}
}

func (rsp *SEN08942RainSensorProvider) resetPinHighs() {
	rsp.pinLock.Lock()
	rsp.highCounts = 0
	rsp.pinLock.Unlock()
}

func (rsp *SEN08942RainSensorProvider) resetRainfall() {
	rsp.rainfallLock.Lock()
	rsp.totalRainfall = 0
	rsp.rainfallLock.Unlock()
}

func (rsp *SEN08942RainSensorProvider) monitorRainfall() {
	for {
		rsp.resetPinHighs()
		select {
		case <-rsp.haltCh:
			rsp.haltedCh <- struct{}{}
			return
		case <-time.After(rsp.interval):
		}

		rsp.pinLock.Lock()
		highs := rsp.highCounts
		rsp.pinLock.Unlock()

		rainfall := float64(highs) * RainfallMMPerTip

		rsp.rainfallLock.Lock()
		rsp.totalRainfall += rainfall
		rsp.rainfallLock.Unlock()
	}
}

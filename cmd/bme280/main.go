package main

import (
	"fmt"

	"github.com/maciej/bme280"
	"golang.org/x/exp/io/i2c"
)

const i2caddr = 0x76

func main() {
	device, err := i2c.Open(&i2c.Devfs{Dev: "/dev/i2c-1"}, i2caddr)
	if err != nil {
		panic(err)
	}

	driver := bme280.New(device)
	err = driver.InitWith(bme280.ModeForced, bme280.Settings{
		Filter:                  bme280.FilterOff,
		PressureOversampling:    bme280.Oversampling1x,
		TemperatureOversampling: bme280.Oversampling1x,
		HumidityOversampling:    bme280.Oversampling1x,
	})

	response, err := driver.Read()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Humidity: %f, Temperature: %f, Pressure: %f\n", response.Humidity, response.Temperature,
		response.Pressure)
}

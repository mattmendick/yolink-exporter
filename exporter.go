package main

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

type YoLinkExporter struct {
	client *YoLinkClient
	mutex  sync.RWMutex

	// Metrics
	temperature *prometheus.Desc
	humidity    *prometheus.Desc
	battery     *prometheus.Desc
	online      *prometheus.Desc
	lastUpdated *prometheus.Desc
	up          *prometheus.Desc

	// Cache
	lastScrape   time.Time
	devices      []Device
	deviceStates map[string]*DeviceStateResponse
}

func NewYoLinkExporter(client *YoLinkClient) *YoLinkExporter {
	return &YoLinkExporter{
		client: client,
		temperature: prometheus.NewDesc(
			"yolink_temperature_celsius",
			"Temperature in Celsius",
			[]string{"device_id", "device_name", "model"},
			nil,
		),
		humidity: prometheus.NewDesc(
			"yolink_humidity_percent",
			"Humidity percentage",
			[]string{"device_id", "device_name", "model"},
			nil,
		),
		battery: prometheus.NewDesc(
			"yolink_battery_level",
			"Battery level (1-4)",
			[]string{"device_id", "device_name", "model"},
			nil,
		),
		online: prometheus.NewDesc(
			"yolink_device_online",
			"Device online status (1=online, 0=offline)",
			[]string{"device_id", "device_name", "model"},
			nil,
		),
		lastUpdated: prometheus.NewDesc(
			"yolink_last_updated_timestamp",
			"Unix timestamp of when the device last reported data",
			[]string{"device_id", "device_name", "model"},
			nil,
		),
		up: prometheus.NewDesc(
			"yolink_up",
			"Whether the YoLink exporter is working (1) or not (0)",
			nil,
			nil,
		),
		deviceStates: make(map[string]*DeviceStateResponse),
	}
}

func (e *YoLinkExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.temperature
	ch <- e.humidity
	ch <- e.battery
	ch <- e.online
	ch <- e.lastUpdated
	ch <- e.up
}

func (e *YoLinkExporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Check if we need to refresh data
	if time.Since(e.lastScrape) > time.Duration(viper.GetInt("scrape.interval"))*time.Second {
		if err := e.refreshData(); err != nil {
			log.Printf("Failed to refresh data: %v", err)
			ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 0)
			return
		}
		e.lastScrape = time.Now()
	}

	// Export up metric
	ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 1)

	// Export device metrics
	for _, device := range e.devices {
		state, exists := e.deviceStates[device.DeviceID]
		if !exists {
			continue
		}

		labels := []string{device.DeviceID, device.Name, device.ModelName}

		// Online status
		onlineValue := 0.0
		if state.Data.Online {
			onlineValue = 1.0
		}
		ch <- prometheus.MustNewConstMetric(e.online, prometheus.GaugeValue, onlineValue, labels...)

		// Last updated timestamp
		if reportAt, err := time.Parse(time.RFC3339, state.Data.ReportAt); err == nil {
			lastUpdatedValue := float64(reportAt.Unix())
			ch <- prometheus.MustNewConstMetric(e.lastUpdated, prometheus.GaugeValue, lastUpdatedValue, labels...)
		} else {
			log.Printf("Failed to parse reportAt time for device %s: %v", device.DeviceID, err)
		}

		// Only export sensor data if device is online
		if state.Data.Online {
			// Temperature
			ch <- prometheus.MustNewConstMetric(e.temperature, prometheus.GaugeValue, state.Data.State.Temperature, labels...)

			// Humidity
			ch <- prometheus.MustNewConstMetric(e.humidity, prometheus.GaugeValue, state.Data.State.Humidity, labels...)

			// Battery level
			ch <- prometheus.MustNewConstMetric(e.battery, prometheus.GaugeValue, float64(state.Data.State.Battery), labels...)
		}
	}
}

func (e *YoLinkExporter) refreshData() error {
	// Get device list
	devices, err := e.client.GetDevices()
	if err != nil {
		return err
	}

	e.devices = devices
	e.deviceStates = make(map[string]*DeviceStateResponse)

	// Get state for each device
	for _, device := range devices {
		state, err := e.client.GetDeviceState(device)
		if err != nil {
			log.Printf("Failed to get state for device %s (%s): %v", device.Name, device.DeviceID, err)
			continue
		}
		e.deviceStates[device.DeviceID] = state
	}

	return nil
}

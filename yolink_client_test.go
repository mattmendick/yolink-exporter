package main

import (
	"testing"
	"time"
)

func TestNewYoLinkClient(t *testing.T) {
	client := NewYoLinkClient("test-key", "test-secret", "https://api.yosmart.com")

	if client.apiKey != "test-key" {
		t.Errorf("Expected apiKey to be 'test-key', got '%s'", client.apiKey)
	}

	if client.secret != "test-secret" {
		t.Errorf("Expected secret to be 'test-secret', got '%s'", client.secret)
	}

	if client.endpoint != "https://api.yosmart.com" {
		t.Errorf("Expected endpoint to be 'https://api.yosmart.com', got '%s'", client.endpoint)
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}

func TestTokenExpiry(t *testing.T) {
	client := NewYoLinkClient("test-key", "test-secret", "https://api.yosmart.com")

	// Test with no token
	if !client.tokenExpiry.IsZero() {
		t.Error("Expected tokenExpiry to be zero when no token is set")
	}

	// Test token expiry logic
	client.tokenExpiry = time.Now().Add(-1 * time.Second) // Expired
	if !time.Now().After(client.tokenExpiry) {
		t.Error("Expected token to be expired")
	}

	client.tokenExpiry = time.Now().Add(1 * time.Hour) // Valid
	if time.Now().After(client.tokenExpiry) {
		t.Error("Expected token to be valid")
	}
}

func TestDeviceFiltering(t *testing.T) {
	devices := []Device{
		{
			DeviceID:  "test1",
			Name:      "Test Sensor 1",
			Type:      "THSensor",
			ModelName: "YS8007-UC",
		},
		{
			DeviceID:  "test2",
			Name:      "Test Hub",
			Type:      "Hub",
			ModelName: "YS1603-UC",
		},
		{
			DeviceID:  "test3",
			Name:      "Test Sensor 2",
			Type:      "THSensor",
			ModelName: "YS8007-UC",
		},
	}

	// Simulate filtering logic
	var thSensors []Device
	for _, device := range devices {
		if device.Type == "THSensor" && device.ModelName == "YS8007-UC" {
			thSensors = append(thSensors, device)
		}
	}

	if len(thSensors) != 2 {
		t.Errorf("Expected 2 THSensor devices, got %d", len(thSensors))
	}

	if thSensors[0].DeviceID != "test1" {
		t.Errorf("Expected first device to be 'test1', got '%s'", thSensors[0].DeviceID)
	}

	if thSensors[1].DeviceID != "test3" {
		t.Errorf("Expected second device to be 'test3', got '%s'", thSensors[1].DeviceID)
	}
}

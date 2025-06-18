package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type YoLinkClient struct {
	apiKey       string
	secret       string
	endpoint     string
	accessToken  string
	refreshToken string
	tokenExpiry  time.Time
	httpClient   *http.Client
}

type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int      `json:"expires_in"`
	RefreshToken string   `json:"refresh_token"`
	Scope        []string `json:"scope"`
}

type DeviceListResponse struct {
	Code string `json:"code"`
	Time int64  `json:"time"`
	Data struct {
		Devices []Device `json:"devices"`
	} `json:"data"`
}

type Device struct {
	DeviceID       string `json:"deviceId"`
	DeviceUDID     string `json:"deviceUDID"`
	Name           string `json:"name"`
	Token          string `json:"token"`
	Type           string `json:"type"`
	ParentDeviceID string `json:"parentDeviceId"`
	ModelName      string `json:"modelName"`
	ServiceZone    string `json:"serviceZone"`
}

type DeviceStateResponse struct {
	Code string `json:"code"`
	Time int64  `json:"time"`
	Data struct {
		Online bool `json:"online"`
		State  struct {
			Battery     int     `json:"battery"`
			Humidity    float64 `json:"humidity"`
			Temperature float64 `json:"temperature"`
			State       string  `json:"state"`
		} `json:"state"`
		DeviceID string `json:"deviceId"`
		ReportAt string `json:"reportAt"`
	} `json:"data"`
}

type APIRequest struct {
	Method       string `json:"method"`
	Time         int64  `json:"time"`
	TargetDevice string `json:"targetDevice,omitempty"`
	Token        string `json:"token,omitempty"`
}

func NewYoLinkClient(apiKey, secret, endpoint string) *YoLinkClient {
	return &YoLinkClient{
		apiKey:     apiKey,
		secret:     secret,
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *YoLinkClient) ensureValidToken() error {
	if c.accessToken == "" || time.Now().After(c.tokenExpiry) {
		if c.refreshToken != "" {
			return c.refreshAccessToken()
		}
		return c.getInitialToken()
	}
	return nil
}

func (c *YoLinkClient) getInitialToken() error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.apiKey)
	data.Set("client_secret", c.secret)

	req, err := http.NewRequest("POST", c.endpoint+"/open/yolink/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "yolink-exporter/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.refreshToken = tokenResp.RefreshToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second) // Subtract 60 seconds for safety

	return nil
}

func (c *YoLinkClient) refreshAccessToken() error {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", c.apiKey)
	data.Set("refresh_token", c.refreshToken)

	req, err := http.NewRequest("POST", c.endpoint+"/open/yolink/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "yolink-exporter/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse refresh response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.refreshToken = tokenResp.RefreshToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return nil
}

func (c *YoLinkClient) GetDevices() ([]Device, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	reqBody := APIRequest{
		Method: "Home.getDeviceList",
		Time:   time.Now().Unix(),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.endpoint+"/open/yolink/v2/api", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "yolink-exporter/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device list request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deviceResp DeviceListResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, fmt.Errorf("failed to parse device response: %w", err)
	}

	if deviceResp.Code != "000000" {
		return nil, fmt.Errorf("API returned error code: %s", deviceResp.Code)
	}

	// Filter for THSensor devices only
	var thSensors []Device
	for _, device := range deviceResp.Data.Devices {
		if device.Type == "THSensor" && device.ModelName == "YS8007-UC" {
			thSensors = append(thSensors, device)
		}
	}

	return thSensors, nil
}

func (c *YoLinkClient) GetDeviceState(device Device) (*DeviceStateResponse, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	reqBody := APIRequest{
		Method:       "THSensor.getState",
		Time:         time.Now().Unix(),
		TargetDevice: device.DeviceID,
		Token:        device.Token,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.endpoint+"/open/yolink/v2/api", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "yolink-exporter/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get device state: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device state request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var stateResp DeviceStateResponse
	if err := json.Unmarshal(body, &stateResp); err != nil {
		return nil, fmt.Errorf("failed to parse state response: %w", err)
	}

	if stateResp.Code != "000000" {
		return nil, fmt.Errorf("API returned error code: %s", stateResp.Code)
	}

	return &stateResp, nil
}

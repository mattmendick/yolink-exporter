# YoLink Prometheus Exporter

A Prometheus exporter for YoLink thermometer/hygrometer devices that fetches temperature, humidity, and battery data from the YoLink API and exposes it as Prometheus metrics.

> **Note**: This project was built almost exclusively with [Cursor](https://cursor.sh).

## Features

- Fetches data from YoLink API using OAuth2 client credentials flow
- Automatic token refresh handling
- Exports metrics for temperature, humidity, battery level, and device online status
- Supports both binary and Docker deployment
- Configurable via YAML file, environment variables, or CLI flags
- Health check endpoint
- Graceful shutdown handling

## Metrics

The exporter exposes the following Prometheus metrics:

- `yolink_temperature_celsius` - Temperature in Celsius (with labels: device_id, device_name, model)
- `yolink_humidity_percent` - Humidity percentage (with labels: device_id, device_name, model)
- `yolink_battery_level` - Battery level 1-4 (with labels: device_id, device_name, model)
- `yolink_device_online` - Device online status 1=online, 0=offline (with labels: device_id, device_name, model)
- `yolink_last_updated_timestamp` - Unix timestamp of when the device last reported data (with labels: device_id, device_name, model)
- `yolink_up` - Whether the exporter is working (1) or not (0)

## Installation

### Prerequisites

- Go 1.21 or later (for building from source)
- Docker and Docker Compose (for containerized deployment)
- YoLink API credentials (API key and secret)

### Building from Source

1. Clone the repository:
```bash
git clone <repository-url>
cd yolink-exporter
```

2. Install dependencies:
```bash
go mod download
```

3. Build the binary:
```bash
go build -o yolink-exporter .
```

### Running as Binary

1. Create a configuration file `config.yaml`:
```yaml
server:
  host: "0.0.0.0"
  port: 8080

api:
  endpoint: "https://api.yosmart.com"

scrape:
  interval: 60  # seconds
```

2. Run the exporter with your API credentials:
```bash
./yolink-exporter --api-key "your-api-key" --secret "your-secret"
```

Or use environment variables:
```bash
export YOLINK_API_KEY="your-api-key"
export YOLINK_SECRET="your-secret"
./yolink-exporter
```

### Running with Docker

1. Create a `.env` file with your API credentials:
```bash
cp env.example .env
# Edit .env with your actual API credentials
```

2. Build and run with Docker Compose:
```bash
docker-compose up -d
```

Or build and run manually:
```bash
docker build -t yolink-exporter .
docker run -p 8080:8080 --env-file .env yolink-exporter
```

## Configuration

### Configuration File (config.yaml)

```yaml
server:
  host: "0.0.0.0"  # Host to bind to
  port: 8080       # Port to listen on

api:
  endpoint: "https://api.yosmart.com"  # YoLink API endpoint

scrape:
  interval: 60  # How often to fetch data from YoLink API (seconds)
```

### Environment Variables

- `YOLINK_API_KEY` - Your YoLink API key
- `YOLINK_SECRET` - Your YoLink API secret

### CLI Flags

- `--api-key` - YoLink API key
- `--secret` - YoLink API secret
- `--config` - Path to configuration file (default: ./config.yaml)

## API Credentials Priority

The application looks for API credentials in the following order:
1. CLI flags (`--api-key`, `--secret`)
2. Environment variables (`YOLINK_API_KEY`, `YOLINK_SECRET`)
3. Configuration file (`api.key`, `api.secret`)

## Endpoints

- `/metrics` - Prometheus metrics endpoint
- `/health` - Health check endpoint (returns 200 OK)

## Prometheus Configuration

Add the following to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'yolink-exporter'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 30s
```

## Example Queries

### Get temperature for all devices:
```
yolink_temperature_celsius
```

### Get humidity for a specific device:
```
yolink_humidity_percent{device_name="Andrew Thermometer"}
```

### Check if devices are online:
```
yolink_device_online
```

### Get battery levels for devices with low battery:
```
yolink_battery_level < 2
```

### Get last update timestamps for all devices:
```
yolink_last_updated_timestamp
```

### Find devices that haven't reported in the last hour:
```
yolink_last_updated_timestamp < (time() - 3600)
```

### Get the age of the most recent data for each device:
```
time() - yolink_last_updated_timestamp
```

## Troubleshooting

### Check if the exporter is running:
```bash
curl http://localhost:8080/health
```

### Check metrics:
```bash
curl http://localhost:8080/metrics
```

### View logs:
```bash
docker-compose logs -f yolink-exporter
```

### Common Issues

1. **Authentication errors**: Verify your API key and secret are correct
2. **No devices found**: Ensure you have THSensor devices with model YS8007-UC
3. **Connection timeouts**: Check your network connectivity to api.yosmart.com

## Development

### Building for different platforms:
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o yolink-exporter-linux .

# macOS
GOOS=darwin GOARCH=amd64 go build -o yolink-exporter-darwin .

# Windows
GOOS=windows GOARCH=amd64 go build -o yolink-exporter-windows.exe .
```

### Running tests:
```bash
go test ./...
```

## License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details. 
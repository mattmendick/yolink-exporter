package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	apiKey  string
	secret  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "yolink-exporter",
		Short: "Prometheus exporter for YoLink thermometer/hygrometer devices",
		Long:  `A Prometheus exporter that fetches data from YoLink API and exposes metrics for temperature, humidity, and battery levels.`,
		RunE:  run,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "YoLink API key")
	rootCmd.PersistentFlags().StringVar(&secret, "secret", "", "YoLink API secret")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Load configuration
	if err := loadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get API credentials from flags, environment, or config
	apiKey := getAPIKey()
	secret := getSecret()

	if apiKey == "" || secret == "" {
		return fmt.Errorf("API key and secret are required. Use --api-key and --secret flags, or set YOLINK_API_KEY and YOLINK_SECRET environment variables")
	}

	// Create YoLink client
	client := NewYoLinkClient(apiKey, secret, viper.GetString("api.endpoint"))

	// Create exporter
	exporter := NewYoLinkExporter(client)

	// Register metrics
	prometheus.MustRegister(exporter)

	// Setup HTTP server
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", viper.GetString("server.host"), viper.GetInt("server.port"))
	log.Printf("Starting YoLink exporter on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: nil,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited")
	return nil
}

func getAPIKey() string {
	// Priority: CLI flag > environment variable > config file
	if apiKey != "" {
		return apiKey
	}
	if envKey := os.Getenv("YOLINK_API_KEY"); envKey != "" {
		return envKey
	}
	return viper.GetString("api.key")
}

func getSecret() string {
	// Priority: CLI flag > environment variable > config file
	if secret != "" {
		return secret
	}
	if envSecret := os.Getenv("YOLINK_SECRET"); envSecret != "" {
		return envSecret
	}
	return viper.GetString("api.secret")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	// Set defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("api.endpoint", "https://api.yosmart.com")
	viper.SetDefault("scrape.interval", 60)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		log.Println("No config file found, using defaults")
	}

	return nil
}

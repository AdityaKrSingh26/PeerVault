package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr     string        `yaml:"listen_addr"`
	AdvertiseAddr  string        `yaml:"advertise_addr"`
	Bootstrap      []string      `yaml:"bootstrap"`
	Interactive    bool          `yaml:"interactive"`
	Demo           bool          `yaml:"demo"`
	EncKey         string        `yaml:"enc_key"`
	DetectPublicIP bool          `yaml:"detect_public_ip"`
	Verbose        bool          `yaml:"verbose"`
	Debug          bool          `yaml:"debug"`
	MetricsAddr    string        `yaml:"metrics_addr"`
	DiscoverLocal  bool          `yaml:"discover_local"`
	DiscoverPex    bool          `yaml:"discover_pex"`
	QuotaSize      string        `yaml:"quota"`
	LogLevel       string        `yaml:"log_level"`
	FetchTimeout   time.Duration `yaml:"fetch_timeout"`
	PexInterval    time.Duration `yaml:"pex_interval"`
	GCInterval     time.Duration `yaml:"gc_interval"`
	GCDelay        time.Duration `yaml:"gc_delay"`
}

func DefaultConfig() *Config {
	return &Config{
		ListenAddr:   ":3000",
		LogLevel:     "info",
		FetchTimeout: 5 * time.Second,
		PexInterval:  5 * time.Minute,
		GCInterval:   1 * time.Hour,
		GCDelay:      5 * time.Minute,
	}
}

func (cfg *Config) LoadFromEnv() {
	if val, ok := os.LookupEnv("PEERVAULT_LISTEN"); ok {
		cfg.ListenAddr = val
	}
	if val, ok := os.LookupEnv("PEERVAULT_ADVERTISE"); ok {
		cfg.AdvertiseAddr = val
	}
	if val, ok := os.LookupEnv("PEERVAULT_BOOTSTRAP"); ok {
		parts := strings.Split(val, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		cfg.Bootstrap = parts
	}
	if val, ok := os.LookupEnv("PEERVAULT_INTERACTIVE"); ok {
		cfg.Interactive = strings.ToLower(val) == "true" || val == "1"
	}
	if val, ok := os.LookupEnv("PEERVAULT_DEMO"); ok {
		cfg.Demo = strings.ToLower(val) == "true" || val == "1"
	}
	if val, ok := os.LookupEnv("PEERVAULT_ENC_KEY"); ok {
		cfg.EncKey = val
	} else if val, ok := os.LookupEnv("PEERVAULT_KEY"); ok {
		cfg.EncKey = val
	}
	if val, ok := os.LookupEnv("PEERVAULT_PUBLIC_IP"); ok {
		cfg.DetectPublicIP = strings.ToLower(val) == "true" || val == "1"
	}
	if val, ok := os.LookupEnv("PEERVAULT_VERBOSE"); ok {
		cfg.Verbose = strings.ToLower(val) == "true" || val == "1"
	}
	if val, ok := os.LookupEnv("PEERVAULT_DEBUG"); ok {
		cfg.Debug = strings.ToLower(val) == "true" || val == "1"
	}
	if val, ok := os.LookupEnv("PEERVAULT_METRICS"); ok {
		cfg.MetricsAddr = val
	}
	if val, ok := os.LookupEnv("PEERVAULT_DISCOVER_LOCAL"); ok {
		cfg.DiscoverLocal = strings.ToLower(val) == "true" || val == "1"
	}
	if val, ok := os.LookupEnv("PEERVAULT_DISCOVER_PEX"); ok {
		cfg.DiscoverPex = strings.ToLower(val) == "true" || val == "1"
	}
	if val, ok := os.LookupEnv("PEERVAULT_QUOTA"); ok {
		cfg.QuotaSize = val
	}
	if val, ok := os.LookupEnv("PEERVAULT_LOG_LEVEL"); ok {
		cfg.LogLevel = val
	}
	if val, ok := os.LookupEnv("PEERVAULT_FETCH_TIMEOUT"); ok {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.FetchTimeout = d
		}
	}
	if val, ok := os.LookupEnv("PEERVAULT_PEX_INTERVAL"); ok {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.PexInterval = d
		}
	}
	if val, ok := os.LookupEnv("PEERVAULT_GC_INTERVAL"); ok {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.GCInterval = d
		}
	}
	if val, ok := os.LookupEnv("PEERVAULT_GC_DELAY"); ok {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.GCDelay = d
		}
	}
}

func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Define command-line flags
	configPath := flag.String("config", "", "Path to YAML config file")
	listenAddr := flag.String("addr", "", "Listen address")
	advertiseAddr := flag.String("advertise", "", "Address to advertise to peers")
	bootstrap := flag.String("bootstrap", "", "Bootstrap nodes (comma-separated)")
	interactive := flag.Bool("interactive", false, "Run in interactive mode")
	demo := flag.Bool("demo", false, "Run demo mode")
	encKey := flag.String("key", "", "Encryption key (32 bytes)")
	detectPublicIP := flag.Bool("public-ip", false, "Auto-detect public IP")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	debug := flag.Bool("debug", false, "Enable debug mode")
	metricsAddr := flag.String("metrics", "", "Metrics server address")
	discoverLocal := flag.Bool("discover-local", false, "Enable local discovery")
	discoverPex := flag.Bool("discover-pex", false, "Enable peer exchange")
	quotaSize := flag.String("quota", "", "Storage quota size")
	logLevel := flag.String("log-level", "", "Log level")
	fetchTimeout := flag.Duration("fetch-timeout", 0, "Fetch timeout")
	pexInterval := flag.Duration("pex-interval", 0, "PEX interval")
	gcInterval := flag.Duration("gc-interval", 0, "GC interval")
	gcDelay := flag.Duration("gc-delay", 0, "GC delay")

	flag.Parse()

	// 1. YAML Config File
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse yaml config: %w", err)
		}
	}

	// 2. Env Vars
	cfg.LoadFromEnv()

	// 3. CLI flags (overrides everything if explicitly set)
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if setFlags["addr"] {
		cfg.ListenAddr = *listenAddr
	}
	if setFlags["advertise"] {
		cfg.AdvertiseAddr = *advertiseAddr
	}
	if setFlags["bootstrap"] {
		parts := strings.Split(*bootstrap, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		cfg.Bootstrap = parts
	}
	if setFlags["interactive"] {
		cfg.Interactive = *interactive
	}
	if setFlags["demo"] {
		cfg.Demo = *demo
	}
	if setFlags["key"] {
		cfg.EncKey = *encKey
	}
	if setFlags["public-ip"] {
		cfg.DetectPublicIP = *detectPublicIP
	}
	if setFlags["verbose"] {
		cfg.Verbose = *verbose
	}
	if setFlags["debug"] {
		cfg.Debug = *debug
	}
	if setFlags["metrics"] {
		cfg.MetricsAddr = *metricsAddr
	}
	if setFlags["discover-local"] {
		cfg.DiscoverLocal = *discoverLocal
	}
	if setFlags["discover-pex"] {
		cfg.DiscoverPex = *discoverPex
	}
	if setFlags["quota"] {
		cfg.QuotaSize = *quotaSize
	}
	if setFlags["log-level"] {
		cfg.LogLevel = *logLevel
	}
	if setFlags["fetch-timeout"] {
		cfg.FetchTimeout = *fetchTimeout
	}
	if setFlags["pex-interval"] {
		cfg.PexInterval = *pexInterval
	}
	if setFlags["gc-interval"] {
		cfg.GCInterval = *gcInterval
	}
	if setFlags["gc-delay"] {
		cfg.GCDelay = *gcDelay
	}

	return cfg, nil
}

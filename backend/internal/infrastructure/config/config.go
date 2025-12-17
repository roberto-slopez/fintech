package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config estructura principal de configuración
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Queue    QueueConfig    `mapstructure:"queue"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Webhook  WebhookConfig  `mapstructure:"webhook"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig configuración del servidor HTTP
type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	Mode            string        `mapstructure:"mode"` // debug, release, test
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	CorsOrigins     []string      `mapstructure:"cors_origins"`
}

// DatabaseConfig configuración de la base de datos
type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	DirectURL       string        `mapstructure:"direct_url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	SSLMode         string        `mapstructure:"ssl_mode"`
}

// CacheConfig configuración del cache
type CacheConfig struct {
	Type     string `mapstructure:"type"` // redis, memory
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	TTL      int    `mapstructure:"ttl"` // TTL por defecto en segundos
}

// QueueConfig configuración de la cola de trabajos
type QueueConfig struct {
	Type           string        `mapstructure:"type"` // postgres, redis
	WorkerCount    int           `mapstructure:"worker_count"`
	PollInterval   time.Duration `mapstructure:"poll_interval"`
	MaxRetries     int           `mapstructure:"max_retries"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
	JobTimeout     time.Duration `mapstructure:"job_timeout"`
}

// JWTConfig configuración de JWT
type JWTConfig struct {
	Secret           string        `mapstructure:"secret"`
	AccessExpiry     time.Duration `mapstructure:"access_expiry"`
	RefreshExpiry    time.Duration `mapstructure:"refresh_expiry"`
	Issuer           string        `mapstructure:"issuer"`
}

// WebhookConfig configuración de webhooks
type WebhookConfig struct {
	Secret         string        `mapstructure:"secret"`
	Timeout        time.Duration `mapstructure:"timeout"`
	MaxRetries     int           `mapstructure:"max_retries"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
	CallbackURL    string        `mapstructure:"callback_url"`
}

// LogConfig configuración de logging
type LogConfig struct {
	Level      string `mapstructure:"level"` // debug, info, warn, error
	Format     string `mapstructure:"format"` // json, console
	Output     string `mapstructure:"output"` // stdout, file
	FilePath   string `mapstructure:"file_path"`
}

// Load carga la configuración desde archivo y variables de entorno
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/fintech/")
	
	// Variables de entorno
	viper.SetEnvPrefix("FINTECH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	
	// Valores por defecto
	setDefaults()
	
	// Mapeo de variables de entorno específicas
	viper.BindEnv("database.url", "DATABASE_URL")
	viper.BindEnv("database.direct_url", "DATABASE_URL_DIRECT")
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	viper.BindEnv("cache.host", "REDIS_HOST")
	viper.BindEnv("cache.port", "REDIS_PORT")
	viper.BindEnv("cache.password", "REDIS_PASSWORD")
	
	// Intentar leer archivo de configuración
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}
	
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}
	
	// Validar configuración requerida
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	
	return &cfg, nil
}

// setDefaults establece valores por defecto
func setDefaults() {
	// Server
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", 15*time.Second)
	viper.SetDefault("server.write_timeout", 15*time.Second)
	viper.SetDefault("server.shutdown_timeout", 30*time.Second)
	viper.SetDefault("server.cors_origins", []string{"http://localhost:5173", "http://localhost:3000"})
	
	// Database
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.conn_max_lifetime", 30*time.Minute)
	viper.SetDefault("database.conn_max_idle_time", 5*time.Minute)
	viper.SetDefault("database.ssl_mode", "require")
	
	// Cache
	viper.SetDefault("cache.type", "memory")
	viper.SetDefault("cache.host", "localhost")
	viper.SetDefault("cache.port", 6379)
	viper.SetDefault("cache.db", 0)
	viper.SetDefault("cache.ttl", 300)
	
	// Queue
	viper.SetDefault("queue.type", "postgres")
	viper.SetDefault("queue.worker_count", 5)
	viper.SetDefault("queue.poll_interval", 1*time.Second)
	viper.SetDefault("queue.max_retries", 3)
	viper.SetDefault("queue.retry_delay", 30*time.Second)
	viper.SetDefault("queue.job_timeout", 5*time.Minute)
	
	// JWT
	viper.SetDefault("jwt.secret", "change-me-in-production")
	viper.SetDefault("jwt.access_expiry", 15*time.Minute)
	viper.SetDefault("jwt.refresh_expiry", 7*24*time.Hour)
	viper.SetDefault("jwt.issuer", "fintech-multipass")
	
	// Webhook
	viper.SetDefault("webhook.timeout", 30*time.Second)
	viper.SetDefault("webhook.max_retries", 3)
	viper.SetDefault("webhook.retry_delay", 5*time.Second)
	
	// Log
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
}

// Validate valida la configuración
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}
	
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	
	return nil
}


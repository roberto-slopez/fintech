package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/config"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// CacheService interface para operaciones de caché
type CacheService interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Close() error

	// Métodos específicos para el dominio
	GetApplication(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error)
	SetApplication(ctx context.Context, app *entity.CreditApplication) error
	InvalidateApplication(ctx context.Context, id uuid.UUID) error

	GetCountry(ctx context.Context, code string) (*entity.Country, error)
	SetCountry(ctx context.Context, country *entity.Country) error
	GetAllCountries(ctx context.Context) ([]entity.Country, error)
	SetAllCountries(ctx context.Context, countries []entity.Country) error
}

// Prefijos de cache
const (
	prefixApplication = "app:"
	prefixCountry     = "country:"
	prefixCountries   = "countries:all"
	prefixUser        = "user:"
	prefixRules       = "rules:"
)

// TTLs por defecto (en segundos)
const (
	ttlApplication = 300  // 5 minutos
	ttlCountry     = 3600 // 1 hora
	ttlRules       = 1800 // 30 minutos
)

// RedisCache implementación de caché con Redis
type RedisCache struct {
	client     *redis.Client
	defaultTTL int
}

// NewRedisCache crea una nueva instancia de cache con Redis
func NewRedisCache(cfg config.CacheConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:     client,
		defaultTTL: cfg.TTL,
	}, nil
}

func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return err
	}
	return json.Unmarshal(data, dest)
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ttl := time.Duration(ttlSeconds) * time.Second
	if ttlSeconds <= 0 {
		ttl = time.Duration(c.defaultTTL) * time.Second
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	return result > 0, err
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Métodos específicos del dominio
func (c *RedisCache) GetApplication(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error) {
	var app entity.CreditApplication
	err := c.Get(ctx, prefixApplication+id.String(), &app)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (c *RedisCache) SetApplication(ctx context.Context, app *entity.CreditApplication) error {
	return c.Set(ctx, prefixApplication+app.ID.String(), app, ttlApplication)
}

func (c *RedisCache) InvalidateApplication(ctx context.Context, id uuid.UUID) error {
	return c.Delete(ctx, prefixApplication+id.String())
}

func (c *RedisCache) GetCountry(ctx context.Context, code string) (*entity.Country, error) {
	var country entity.Country
	err := c.Get(ctx, prefixCountry+code, &country)
	if err != nil {
		return nil, err
	}
	return &country, nil
}

func (c *RedisCache) SetCountry(ctx context.Context, country *entity.Country) error {
	return c.Set(ctx, prefixCountry+country.Code, country, ttlCountry)
}

func (c *RedisCache) GetAllCountries(ctx context.Context) ([]entity.Country, error) {
	var countries []entity.Country
	err := c.Get(ctx, prefixCountries, &countries)
	if err != nil {
		return nil, err
	}
	return countries, nil
}

func (c *RedisCache) SetAllCountries(ctx context.Context, countries []entity.Country) error {
	return c.Set(ctx, prefixCountries, countries, ttlCountry)
}

// MemoryCache implementación de caché en memoria (fallback)
type MemoryCache struct {
	data       map[string]cacheItem
	mu         sync.RWMutex
	defaultTTL int
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache crea una nueva instancia de cache en memoria
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		data:       make(map[string]cacheItem),
		defaultTTL: 300,
	}
	// Iniciar limpieza periódica
	go cache.cleanup()
	return cache
}

func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.data {
			if now.After(item.expiresAt) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}

func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.data[key]
	if !exists || time.Now().After(item.expiresAt) {
		return fmt.Errorf("key not found: %s", key)
	}
	return json.Unmarshal(item.value, dest)
}

func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	ttl := ttlSeconds
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheItem{
		value:     data,
		expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}
	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.data[key]
	if !exists || time.Now().After(item.expiresAt) {
		return false, nil
	}
	return true, nil
}

func (c *MemoryCache) Close() error {
	return nil
}

// Métodos específicos del dominio para MemoryCache
func (c *MemoryCache) GetApplication(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error) {
	var app entity.CreditApplication
	err := c.Get(ctx, prefixApplication+id.String(), &app)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (c *MemoryCache) SetApplication(ctx context.Context, app *entity.CreditApplication) error {
	return c.Set(ctx, prefixApplication+app.ID.String(), app, ttlApplication)
}

func (c *MemoryCache) InvalidateApplication(ctx context.Context, id uuid.UUID) error {
	return c.Delete(ctx, prefixApplication+id.String())
}

func (c *MemoryCache) GetCountry(ctx context.Context, code string) (*entity.Country, error) {
	var country entity.Country
	err := c.Get(ctx, prefixCountry+code, &country)
	if err != nil {
		return nil, err
	}
	return &country, nil
}

func (c *MemoryCache) SetCountry(ctx context.Context, country *entity.Country) error {
	return c.Set(ctx, prefixCountry+country.Code, country, ttlCountry)
}

func (c *MemoryCache) GetAllCountries(ctx context.Context) ([]entity.Country, error) {
	var countries []entity.Country
	err := c.Get(ctx, prefixCountries, &countries)
	if err != nil {
		return nil, err
	}
	return countries, nil
}

func (c *MemoryCache) SetAllCountries(ctx context.Context, countries []entity.Country) error {
	return c.Set(ctx, prefixCountries, countries, ttlCountry)
}


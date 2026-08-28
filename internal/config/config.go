package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddress, DatabasePath string
	MaxBodyBytes                int64
	ShutdownTimeout             time.Duration
}

func FromEnvironment(get func(string) string) (Config, error) {
	c := Config{ListenAddress: ":8080", DatabasePath: "internalauth.db", MaxBodyBytes: 1 << 20, ShutdownTimeout: 5 * time.Second}
	if v := get("APISIX_LISTEN"); v != "" {
		c.ListenAddress = v
	}
	if v := get("APISIX_DB"); v != "" {
		c.DatabasePath = v
	}
	if v := get("APISIX_MAX_BODY"); v != "" {
		n, e := strconv.ParseInt(v, 10, 64)
		if e != nil {
			return c, e
		}
		c.MaxBodyBytes = n
	}
	return c, c.Validate()
}
func (c Config) Validate() error {
	if c.ListenAddress == "" || c.DatabasePath == "" {
		return errors.New("listen address and database path required")
	}
	if c.MaxBodyBytes < 1 {
		return errors.New("max body bytes must be positive")
	}
	return nil
}
func Default() Config { c, _ := FromEnvironment(os.Getenv); return c }

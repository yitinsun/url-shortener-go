package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"url-shortener/apperror"
)

type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
	EnvTest Env = "test"
)

// Config is the process environment. The password is not exported.
type Config struct {
	Env            Env
	BaseURL        string
	BindAddr       string
	DBHost         string
	DBPort         int
	DBName         string
	DBUser         string
	ApplySchema    bool
	OpenAPIEnabled bool
	dbPassword     string
}

func FromEnv() (Config, error) {
	return FromLookup(os.Getenv)
}

func FromLookup(lookup func(string) string) (Config, error) {
	env := EnvDev
	switch lookup("APP_ENV") {
	case "prod":
		env = EnvProd
	case "test":
		env = EnvTest
	case "", "dev":
		env = EnvDev
	default:
		env = EnvDev
	}

	baseURL, err := requiredOr(env, lookup("APP_BASE_URL"), "APP_BASE_URL", "http://localhost:8080")
	if err != nil {
		return Config{}, err
	}
	baseURL = strings.TrimRight(baseURL, "/")

	bindAddr := nonempty(lookup("BIND_ADDR"))
	if bindAddr == "" {
		bindAddr = "0.0.0.0:8080"
	}
	dbHost, err := requiredOr(env, lookup("DB_HOST"), "DB_HOST", "localhost")
	if err != nil {
		return Config{}, err
	}
	portText := nonempty(lookup("DB_PORT"))
	if portText == "" {
		portText = "3306"
	}
	dbPort, err := strconv.Atoi(portText)
	if err != nil || dbPort < 1 || dbPort > 65535 {
		return Config{}, apperror.Unavailable("DB_PORT is not a number")
	}
	dbName := nonempty(lookup("DB_NAME"))
	if dbName == "" {
		if env == EnvProd {
			dbName = "shortener_prod"
		} else {
			dbName = "shortener_dev"
		}
	}
	dbUser, err := requiredOr(env, lookup("DB_USER"), "DB_USER", "root")
	if err != nil {
		return Config{}, err
	}
	var dbPassword string
	if env == EnvProd {
		dbPassword = nonempty(lookup("DB_PASSWORD"))
		if dbPassword == "" {
			return Config{}, apperror.Unavailable("DB_PASSWORD is required in prod")
		}
	} else {
		dbPassword = nonempty(lookup("DB_PASSWORD"))
		if dbPassword == "" {
			dbPassword = "Tiger"
		}
	}

	return Config{
		Env:            env,
		BaseURL:        baseURL,
		BindAddr:       bindAddr,
		DBHost:         dbHost,
		DBPort:         dbPort,
		DBName:         dbName,
		DBUser:         dbUser,
		ApplySchema:    env != EnvProd,
		OpenAPIEnabled: openAPIEnabled(env, lookup("OPENAPI_ENABLED")),
		dbPassword:     dbPassword,
	}, nil
}

func openAPIEnabled(env Env, raw string) bool {
	switch strings.ToLower(nonempty(raw)) {
	case "true":
		return true
	case "false":
		return false
	case "":
		return env != EnvProd
	default:
		return false
	}
}

// DatabaseDSN is the application database. BootstrapDSN has no schema selected.
func (c Config) DatabaseDSN() string { return c.dsn(c.DBName) }

func (c Config) BootstrapDSN() string { return c.dsn("") }

func (c Config) dsn(database string) string {
	path := "/"
	if database != "" {
		path += database
	}
	return fmt.Sprintf("%s:%s@tcp(%s)%s?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		encodeUserinfo(c.DBUser),
		encodeUserinfo(c.dbPassword),
		net.JoinHostPort(c.DBHost, strconv.Itoa(c.DBPort)),
		path,
	)
}

func encodeUserinfo(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch {
		case ch >= 'A' && ch <= 'Z', ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9', ch == '-', ch == '_', ch == '.', ch == '~':
			b.WriteByte(ch)
		default:
			fmt.Fprintf(&b, "%%%02X", ch)
		}
	}
	return b.String()
}

func nonempty(value string) string {
	return strings.TrimSpace(value)
}

func requiredOr(env Env, value string, name string, devDefault string) (string, error) {
	if value = nonempty(value); value != "" {
		return value, nil
	}
	if env == EnvProd {
		return "", apperror.Unavailable(name + " is required in prod")
	}
	return devDefault, nil
}

// BaseHost is the hostname of APP_BASE_URL, without a port.
func (c Config) BaseHost() (string, error) {
	parsed, err := url.ParseRequestURI(c.BaseURL)
	if err != nil {
		return "", apperror.Unavailable("APP_BASE_URL is invalid")
	}
	return parsed.Hostname(), nil
}

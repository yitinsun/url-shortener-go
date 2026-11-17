package config

import (
	"strings"
	"testing"

	"url-shortener/apperror"
)

func TestDevUsesLabDefaults(t *testing.T) {
	cfg, err := FromLookup(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != EnvDev || cfg.BaseURL != "http://localhost:8080" || !cfg.ApplySchema || !cfg.OpenAPIEnabled {
		t.Fatalf("dev config %+v", cfg)
	}
	if cfg.DBName != "shortener_dev" || cfg.DBUser != "root" || cfg.DBPort != 3306 || cfg.BindAddr != "0.0.0.0:8080" {
		t.Fatalf("defaults %+v", cfg)
	}
	dsn := cfg.DatabaseDSN()
	for _, part := range []string{"shortener_dev", "parseTime=true", "loc=UTC", "charset=utf8mb4", "utf8mb4_unicode_ci", "root:Tiger@"} {
		if !strings.Contains(dsn, part) {
			t.Fatalf("dsn %q missing %q", dsn, part)
		}
	}
	if cfg.BootstrapDSN() == dsn || strings.Contains(cfg.BootstrapDSN(), "/shortener_dev") {
		t.Fatalf("bootstrap dsn %q", cfg.BootstrapDSN())
	}
}

func TestProdRejectsAMissingPassword(t *testing.T) {
	_, err := FromLookup(func(name string) string {
		switch name {
		case "APP_ENV":
			return "prod"
		case "APP_BASE_URL":
			return "https://short.example"
		case "DB_HOST":
			return "db"
		case "DB_USER":
			return "app"
		default:
			return ""
		}
	})
	if err == nil || !apperror.IsBadRequest(err) && !strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Fatalf("expected password failure, got %v", err)
	}
}

func TestProdBuildsDSNAndEncodesThePassword(t *testing.T) {
	cfg, err := FromLookup(func(name string) string {
		switch name {
		case "APP_ENV":
			return "prod"
		case "APP_BASE_URL":
			return "https://short.example/"
		case "DB_HOST":
			return "db"
		case "DB_PORT":
			return "3307"
		case "DB_USER":
			return "app"
		case "DB_PASSWORD":
			return "p@ss word"
		case "BIND_ADDR":
			return "127.0.0.1:8080"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != EnvProd || cfg.ApplySchema || cfg.OpenAPIEnabled || cfg.BaseURL != "https://short.example" {
		t.Fatalf("prod config %+v", cfg)
	}
	if cfg.DBName != "shortener_prod" || cfg.DBPort != 3307 || cfg.BindAddr != "127.0.0.1:8080" {
		t.Fatalf("prod fields %+v", cfg)
	}
	dsn := cfg.DatabaseDSN()
	if !strings.Contains(dsn, "p%40ss%20word") || !strings.Contains(dsn, "/shortener_prod") {
		t.Fatalf("dsn %q", dsn)
	}
	if strings.Contains(dsn, "p@ss word") {
		t.Fatalf("password was not encoded: %q", dsn)
	}
}

func TestEmptyValuesFallBackAndABadPortIsRejected(t *testing.T) {
	cfg, err := FromLookup(func(name string) string {
		switch name {
		case "APP_ENV":
			return "test"
		case "APP_BASE_URL", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "BIND_ADDR":
			return "   "
		case "DB_NAME":
			return "shortener_test"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != EnvTest || !cfg.ApplySchema || cfg.BaseURL != "http://localhost:8080" {
		t.Fatalf("test config %+v", cfg)
	}
	if cfg.DBHost != "localhost" || cfg.DBPort != 3306 || cfg.DBName != "shortener_test" || cfg.DBUser != "root" || cfg.BindAddr != "0.0.0.0:8080" {
		t.Fatalf("fallbacks %+v", cfg)
	}
	if !strings.Contains(cfg.DatabaseDSN(), "shortener_test") {
		t.Fatalf("dsn %q", cfg.DatabaseDSN())
	}

	_, err = FromLookup(func(name string) string {
		if name == "DB_PORT" {
			return "nope"
		}
		return ""
	})
	if err == nil || !strings.Contains(err.Error(), "DB_PORT") {
		t.Fatalf("port error %v", err)
	}
}

func TestProdRequiresBaseURLHostAndUser(t *testing.T) {
	for _, missing := range []string{"APP_BASE_URL", "DB_HOST", "DB_USER"} {
		_, err := FromLookup(func(name string) string {
			if name == "APP_ENV" {
				return "prod"
			}
			if name == missing {
				return " "
			}
			switch name {
			case "APP_BASE_URL":
				return "https://short.example"
			case "DB_HOST":
				return "db"
			case "DB_USER":
				return "app"
			case "DB_PASSWORD":
				return "secret"
			default:
				return ""
			}
		})
		if err == nil || !strings.Contains(err.Error(), missing) {
			t.Fatalf("%s: %v", missing, err)
		}
	}
}

func TestUnknownEnvUsesDevDefaults(t *testing.T) {
	cfg, err := FromLookup(func(name string) string {
		if name == "APP_ENV" {
			return "staging"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != EnvDev || !cfg.ApplySchema || !cfg.OpenAPIEnabled || cfg.DBName != "shortener_dev" {
		t.Fatalf("%+v", cfg)
	}
	host, err := cfg.BaseHost()
	if err != nil || host != "localhost" {
		t.Fatalf("host %q %v", host, err)
	}
}

func TestOpenAPIFlagOverridesTheEnvDefault(t *testing.T) {
	enabled, err := FromLookup(func(name string) string {
		switch name {
		case "APP_ENV":
			return "prod"
		case "APP_BASE_URL":
			return "https://short.example"
		case "DB_HOST":
			return "db"
		case "DB_USER":
			return "app"
		case "DB_PASSWORD":
			return "secret"
		case "OPENAPI_ENABLED":
			return "TRUE"
		default:
			return ""
		}
	})
	if err != nil || !enabled.OpenAPIEnabled {
		t.Fatalf("prod override %+v %v", enabled, err)
	}
	disabled, err := FromLookup(func(name string) string {
		if name == "OPENAPI_ENABLED" {
			return "false"
		}
		return ""
	})
	if err != nil || disabled.OpenAPIEnabled {
		t.Fatalf("dev override %+v %v", disabled, err)
	}
}

func TestInvalidBaseURL(t *testing.T) {
	cfg, err := FromLookup(func(name string) string {
		if name == "APP_BASE_URL" {
			return "://bad"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.BaseHost(); err == nil {
		t.Fatal("expected invalid base url")
	}
}

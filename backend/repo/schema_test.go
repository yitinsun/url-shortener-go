package repo

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestSchemaFileHasTheGreenfieldTable(t *testing.T) {
	text, err := loadSchema()
	if err != nil {
		t.Fatal(err)
	}
	createDB, createTable := schemaStatements(text)
	if !strings.Contains(createDB, "shortener_dev") || !strings.Contains(createDB, "utf8mb4") {
		t.Fatalf("database statement %q", createDB)
	}
	for _, part := range []string{"url_mappings", "AUTO_INCREMENT=56800235", "utf8mb4_unicode_ci", "idx_short_code", "idx_long_url"} {
		if !strings.Contains(createTable, part) {
			t.Fatalf("table statement missing %s: %q", part, createTable)
		}
	}
	if strings.Contains(createTable, "fmt.") {
		t.Fatal("schema was assembled with formatting")
	}
}

func TestAlreadyExistsRecognizesMySQLCodes(t *testing.T) {
	if !alreadyExists(&mysql.MySQLError{Number: 1007, Message: "Can't create database"}) {
		t.Fatal("database exists")
	}
	if !alreadyExists(&mysql.MySQLError{Number: 1050, Message: "Table exists"}) {
		t.Fatal("table exists")
	}
	if !alreadyExists(&mysql.MySQLError{Number: 1, Message: "relation already exists"}) {
		t.Fatal("message match")
	}
	if alreadyExists(&mysql.MySQLError{Number: 1045, Message: "Access denied"}) {
		t.Fatal("auth failure is not already exists")
	}
	if alreadyExists(errors.New("connection refused")) {
		t.Fatal("plain error")
	}
}

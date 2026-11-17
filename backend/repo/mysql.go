package repo

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"url-shortener/apperror"
	"url-shortener/config"
)

// DeleteExpiredSQL is the only expiry-delete statement. Both arguments are bound.
const DeleteExpiredSQL = "DELETE FROM url_mappings WHERE expires_at < ? LIMIT ?"

const selectMapping = "SELECT id, long_url, short_code, created_at, expires_at FROM url_mappings "

// URLMapping is one url_mappings row.
type URLMapping struct {
	ID        int64
	LongURL   string
	ShortCode string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Repository is the storage port. Service tests use an in-memory fake.
type Repository interface {
	FindByLongURLPrefix(ctx context.Context, prefix string) ([]URLMapping, error)
	FindByShortCode(ctx context.Context, shortCode string) (*URLMapping, error)
	InsertWithEncodedID(ctx context.Context, longURL string, placeholder string, createdAt time.Time, expiresAt time.Time, encode func(uint64) string) (URLMapping, error)
	DeleteExpiredBefore(ctx context.Context, cutoff time.Time, limit int64) (int64, error)
}

// MySQL is the database/sql implementation.
type MySQL struct {
	db *sql.DB
}

func NewMySQL(db *sql.DB) *MySQL {
	return &MySQL{db: db}
}

func (m *MySQL) FindByLongURLPrefix(ctx context.Context, prefix string) ([]URLMapping, error) {
	rows, err := m.db.QueryContext(ctx, selectMapping+"WHERE LEFT(long_url, 255) = ?", prefix)
	if err != nil {
		return nil, dbErr(err)
	}
	defer rows.Close()
	var out []URLMapping
	for rows.Next() {
		item, err := scanMapping(rows)
		if err != nil {
			return nil, dbErr(err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErr(err)
	}
	return out, nil
}

func (m *MySQL) FindByShortCode(ctx context.Context, shortCode string) (*URLMapping, error) {
	row := m.db.QueryRowContext(ctx, selectMapping+"WHERE short_code = ?", shortCode)
	item, err := scanMapping(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErr(err)
	}
	return &item, nil
}

func (m *MySQL) InsertWithEncodedID(ctx context.Context, longURL string, placeholder string, createdAt time.Time, expiresAt time.Time, encode func(uint64) string) (URLMapping, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return URLMapping{}, dbErr(err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		"INSERT INTO url_mappings (long_url, short_code, created_at, expires_at) VALUES (?, ?, ?, ?)",
		longURL, placeholder, createdAt.UTC(), expiresAt.UTC(),
	)
	if err != nil {
		return URLMapping{}, dbErr(err)
	}
	var id int64
	if err := tx.QueryRowContext(ctx, "SELECT LAST_INSERT_ID()").Scan(&id); err != nil {
		return URLMapping{}, dbErr(err)
	}
	if id < 0 {
		return URLMapping{}, apperror.Unavailable("assigned id does not fit i64")
	}
	shortCode := encode(uint64(id))
	if _, err := tx.ExecContext(ctx, "UPDATE url_mappings SET short_code = ? WHERE id = ?", shortCode, id); err != nil {
		return URLMapping{}, dbErr(err)
	}
	if err := tx.Commit(); err != nil {
		return URLMapping{}, dbErr(err)
	}
	return URLMapping{
		ID:        id,
		LongURL:   longURL,
		ShortCode: shortCode,
		CreatedAt: createdAt.UTC(),
		ExpiresAt: expiresAt.UTC(),
	}, nil
}

func (m *MySQL) DeleteExpiredBefore(ctx context.Context, cutoff time.Time, limit int64) (int64, error) {
	result, err := m.db.ExecContext(ctx, DeleteExpiredSQL, cutoff.UTC(), limit)
	if err != nil {
		return 0, dbErr(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, dbErr(err)
	}
	return affected, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanMapping(row scanner) (URLMapping, error) {
	var item URLMapping
	err := row.Scan(&item.ID, &item.LongURL, &item.ShortCode, &item.CreatedAt, &item.ExpiresAt)
	item.CreatedAt = item.CreatedAt.UTC()
	item.ExpiresAt = item.ExpiresAt.UTC()
	return item, err
}

func dbErr(err error) error {
	if err == nil {
		return nil
	}
	return apperror.Unavailable(err.Error())
}

// Open connects to MySQL. Dev and test apply backend/sql/schema.sql and ignore "already exists".
func Open(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	text, err := loadSchema()
	if err != nil && cfg.ApplySchema {
		return nil, err
	}
	createDB, createTable := schemaStatements(text)
	if cfg.ApplySchema {
		boot, err := openDB(cfg.BootstrapDSN())
		if err != nil {
			return nil, err
		}
		err = execIgnoreExists(ctx, boot, createDB)
		boot.Close()
		if err != nil {
			return nil, err
		}
	}
	db, err := openDB(cfg.DatabaseDSN())
	if err != nil {
		return nil, err
	}
	if cfg.ApplySchema {
		if err := execIgnoreExists(ctx, db, createTable); err != nil {
			db.Close()
			return nil, err
		}
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, apperror.Unavailable(err.Error())
	}
	return db, nil
}

func openDB(dsn string) (*sql.DB, error) {
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, apperror.Unavailable(err.Error())
	}
	base, err := mysql.NewConnector(parsed)
	if err != nil {
		return nil, apperror.Unavailable(err.Error())
	}
	db := sql.OpenDB(sessionConnector{Connector: base})
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(2 * time.Minute)
	return db, nil
}

// sessionConnector sets the MySQL session clock to UTC on every new connection.
type sessionConnector struct {
	driver.Connector
}

func (c sessionConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	execer, ok := conn.(driver.ExecerContext)
	if !ok {
		conn.Close()
		return nil, errors.New("mysql connection cannot set time_zone")
	}
	if _, err := execer.ExecContext(ctx, "SET time_zone = '+00:00'", nil); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func execIgnoreExists(ctx context.Context, db *sql.DB, statement string) error {
	_, err := db.ExecContext(ctx, statement)
	if err == nil || alreadyExists(err) {
		return nil
	}
	return apperror.Unavailable(err.Error())
}

func alreadyExists(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if mysqlErr.Number == 1007 || mysqlErr.Number == 1050 {
			return true
		}
		return strings.Contains(strings.ToLower(mysqlErr.Message), "already exists")
	}
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

func loadSchema() (string, error) {
	for _, candidate := range schemaCandidates() {
		body, err := os.ReadFile(candidate)
		if err == nil {
			return string(body), nil
		}
	}
	return "", apperror.Unavailable("backend/sql/schema.sql was not found")
}

func schemaCandidates() []string {
	candidates := []string{
		filepath.Join("sql", "schema.sql"),
		filepath.Join("backend", "sql", "schema.sql"),
		filepath.Join("..", "sql", "schema.sql"),
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "sql", "schema.sql"))
	}
	return candidates
}

func schemaStatements(text string) (createDB string, createTable string) {
	for _, statement := range strings.Split(text, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		upper := strings.ToUpper(statement)
		switch {
		case strings.HasPrefix(upper, "CREATE DATABASE"):
			createDB = statement
		case strings.HasPrefix(upper, "CREATE TABLE"):
			createTable = statement
		}
	}
	return createDB, createTable
}

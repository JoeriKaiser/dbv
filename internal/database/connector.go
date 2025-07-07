package database

import (
	"database/sql"
	"dbv/internal/schema"
	"dbv/pkg/config"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	// todo add msyql driver
)

type Connector struct {
	db     *sql.DB
	driver string
}

type SchemaExtractor interface {
	ExtractSchema(cfg config.SchemaConfig) (*schema.Schema, error)
}

func NewConnector(databaseURL string) (*Connector, error) {
	driver, dsn, err := ParseDatabaseURL(databaseURL) // Updated call
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connector{
		db:     db,
		driver: driver,
	}, nil
}

func (c *Connector) Close() error {
	return c.db.Close()
}

func (c *Connector) ExtractSchema(cfg config.SchemaConfig) (*schema.Schema, error) {
	var extractor SchemaExtractor

	switch c.driver {
	case "postgres":
		extractor = &PostgreSQLExtractor{db: c.db}
	// case "mysql":
	//     extractor = &MySQLExtractor{db: c.db}
	case "sqlite3":
		extractor = &SQLiteExtractor{db: c.db}
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.driver)
	}

	return extractor.ExtractSchema(cfg)
}

func ParseDatabaseURL(databaseURL string) (driver, dsn string, err error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", "", err
	}

	switch u.Scheme {
	case "postgres", "postgresql":
		return "postgres", databaseURL, nil
	// case "mysql":
	//     dsn = fmt.Sprintf("%s@tcp(%s)%s", u.User.String(), u.Host, u.Path)
	//     if u.RawQuery != "" {
	//         dsn += "?" + u.RawQuery
	//     }
	//     return "mysql", dsn, nil
	case "sqlite", "sqlite3":
		return "sqlite3", strings.TrimPrefix(databaseURL, "sqlite://"), nil
	default:
		return "", "", fmt.Errorf("unsupported database scheme: %s", u.Scheme)
	}
}

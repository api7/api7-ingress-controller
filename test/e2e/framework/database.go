package framework

import (
	_ "embed"
	"os"
	"strings"
)

// DatabaseConfig is the database related configuration entrypoint.
type DatabaseConfig struct {
	DSN string `json:"dsn" yaml:"dsn" mapstructure:"dsn"`

	MaxOpenConns int `json:"max_open_conns" yaml:"max_open_conns" mapstructure:"max_open_conns"`
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns" mapstructure:"max_idle_conns"`
}

type LogOptions struct {
	// Level is the minimum logging level that a logging message should have
	// to output itself.
	Level string `json:"level" yaml:"level"`
	// Output defines the destination file path to output logging messages.
	// Two keywords "stderr" and "stdout" can be specified so that message will
	// be written to stderr or stdout.
	Output string `json:"output" yaml:"output"`
}

func (conf *DatabaseConfig) GetType() string {
	parts := strings.SplitN(conf.DSN, "://", 2)
	if len(parts) > 1 {
		return parts[0]
	}
	return ""
}

var (
	_db string
)

func init() {
	_db = os.Getenv("DB")
	if _db == "" {
		_db = postgres
	}
}

const (
	postgres     = "postgres"
	oceanbase    = "oceanbase"
	mysql        = "mysql"
	postgresDSN  = "postgres://api7ee:changeme@api7-postgresql:5432/api7ee"
	oceanbaseDSN = "mysql://root@tcp(oceanbase:2881)/api7ee"
	mysqlDSN     = "mysql://root:changeme@tcp(mysql:3306)/api7ee"
)

//nolint:unused
func getDSN() string {
	switch _db {
	case postgres:
		return postgresDSN
	case oceanbase:
		return oceanbaseDSN
	case mysql:
		return mysqlDSN
	}
	panic("unknown database")
}

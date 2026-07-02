package sdgorm

import (
	"database/sql"

	"TMA/pkg/sderr"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Address struct {
	// common
	Driver string `json:"driver" toml:"driver"`
	DSN    string `json:"dsn" toml:"dsn"`
	Logger string `json:"logger" toml:"logger"`

	// mysql
	MySqlConn                      gorm.ConnPool `json:"-" toml:"-"`
	MySqlSkipInitializeWithVersion bool          `json:"mysql_skip_initialize_with_version" toml:"mysql_skip_initialize_with_version"`
	MySqlDefaultStringSize         uint          `json:"mysql_default_string_size" toml:"mysql_default_string_size"`
	MySqlDefaultDatetimePrecision  *int          `json:"mysql_default_datetime_precision" toml:"mysql_default_datetime_precision"`
	MySqlDisableDatetimePrecision  bool          `json:"mysql_disable_datetime_precision" toml:"mysql_disable_datetime_precision"`
	MySqlDontSupportRenameIndex    bool          `json:"mysql_dont_support_rename_index" toml:"mysql_dont_support_rename_index"`
	MySqlDontSupportRenameColumn   bool          `json:"mysql_dont_support_rename_column" toml:"mysql_dont_support_rename_column"`
	MySqlDontSupportForShareClause bool          `json:"mysql_dont_support_for_share_clause" toml:"mysql_dont_support_for_share_clause"`

	// postgres
	PostgresConn                 *sql.DB `json:"-" toml:"-"`
	PostgresPreferSimpleProtocol bool    `json:"postgres_prefer_simple_protocol" toml:"postgres_prefer_simple_protocol"`
	PostgresWithoutReturning     bool    `json:"postgres_without_returning" toml:"postgres_without_returning"`
}

var (
	ErrIllegalDriver = sderr.Sentinel("illegal driver")
)

func Dial(addr Address, config *gorm.Config) (*gorm.DB, error) {
	var config1 gorm.Config
	if config != nil {
		config1 = *config
	}
	setupLogger(&config1, &addr)
	switch addr.Driver {
	case "mysql":
		mysqlConfig := mysql.Config{
			DSN:                       addr.DSN,
			Conn:                      addr.MySqlConn,
			SkipInitializeWithVersion: addr.MySqlSkipInitializeWithVersion,
			DefaultStringSize:         addr.MySqlDefaultStringSize,
			DefaultDatetimePrecision:  addr.MySqlDefaultDatetimePrecision,
			DisableDatetimePrecision:  addr.MySqlDisableDatetimePrecision,
			DontSupportRenameIndex:    addr.MySqlDontSupportRenameIndex,
			DontSupportRenameColumn:   addr.MySqlDontSupportRenameColumn,
			DontSupportForShareClause: addr.MySqlDontSupportForShareClause,
		}
		db, err := gorm.Open(mysql.New(mysqlConfig), &config1)
		if err != nil {
			return nil, sderr.WithStack(err)
		}
		return db, nil
	case "sqlite":
		db, err := gorm.Open(sqlite.Open(addr.DSN), &config1)
		if err != nil {
			return nil, sderr.WithStack(err)
		}
		return db, nil
	case "postgres":
		postgresConfig := postgres.Config{
			DSN:                  addr.DSN,
			Conn:                 addr.PostgresConn,
			PreferSimpleProtocol: addr.PostgresPreferSimpleProtocol,
			WithoutReturning:     addr.PostgresWithoutReturning,
		}
		db, err := gorm.Open(postgres.New(postgresConfig), &config1)
		if err != nil {
			return nil, sderr.WithStack(err)
		}
		return db, nil
	default:
		return nil, sderr.WithStack(ErrIllegalDriver)
	}
}

func setupLogger(config *gorm.Config, addr *Address) {
	if config.Logger == nil {
		config.Logger = LoggerOf(addr.Logger)
	}
}

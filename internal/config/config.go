package config

type PgSlowConf struct {
	PGUser          string `mapstructure:"PG_USER"`
	PGPass          string `mapstructure:"PG_PASS"`
	PGHost          string `mapstructure:"PG_HOST"`
	PGPort          string `mapstructure:"PG_PORT"`
	PGDBName        string `mapstructure:"PG_DBNAME"`
	PROMMetricName  string `mapstructure:"PROM_METRIC_NAME"`
	DBScrapPeriodMS uint   `mapstructure:"DB_SCRAP_PERIOD_MS"`
}

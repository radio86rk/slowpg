package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"pgslowquery/internal/pgslowquery"
	"pgslowquery/internal/config"
)

func main() {

	log.SetFormatter(&log.JSONFormatter{})

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	var cfg config.PgSlowConf

	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Error("Error reading config file: ", err)
		return
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Error("Error Unmarshal AppConf: ", err)
		return
	}

	db_conn_str := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", cfg.PGUser, cfg.PGPass, cfg.PGHost, cfg.PGPort, cfg.PGDBName)
	pgpool, err := pgxpool.New(ctx, db_conn_str)
	if err != nil {
		log.Error(err)
		return
	}
	if err = pgpool.Ping(ctx); err != nil {
		log.Error(err)
		return
	}

	go func() {
		if err = pgslowquery.Run(ctx, pgpool, cfg.PROMMetricName, cfg.DBScrapPeriodMS); err != nil {
			log.Error(err)
			cancel()
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Info("Starting exporter :8080")

	go func() {
		if err = http.ListenAndServe(":8080", nil); err != nil {
			log.Error(err)
			cancel()
		}
	}()

	<-ctx.Done()
}

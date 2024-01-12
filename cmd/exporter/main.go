package main

import (
	"pgslowquery/internal/utils"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"fmt"
	"os"
	"context"
	"time"
	"sync"
	"crypto/sha1"
	"net/http"
	//"github.com/google/uuid"

)

type Query struct {
	Query 		string
	State		string
	QueryID 	int64
	QueryStart	time.Time
	StateChange time.Time
	Duration	time.Duration
	IsDelete	bool
}


type QueryCollector struct {
    metric *prometheus.Desc
}

func (c *QueryCollector) Describe(ch chan<- *prometheus.Desc) {
    ch <- c.metric
}

func (c *QueryCollector) Collect(ch chan<- prometheus.Metric) {
	for k,v := range all_queries {
		t := v.QueryStart
		s := prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(c.metric, prometheus.CounterValue, float64(v.Duration),v.Query ))
		ch <- s
		if v.IsDelete {
			delete(all_queries,k)
		}
	}
}


var all_queries map[string]*Query = make(map[string]*Query,100)



func DoQuery(ctx context.Context, pgpool *pgxpool.Pool) (error) {
	query := "SELECT query,state,state_change,query_id, query_start as ts FROM pg_stat_activity WHERE query IS NOT NULL AND query_id IS NOT NULL"
	rows, err := pgpool.Query(ctx, query)
	if err != nil {
		return err
	}
	for rows.Next() {
		q := Query{}
		err = rows.Scan(&q.Query,&q.State,&q.StateChange,&q.QueryID,&q.QueryStart)
		if err != nil {
			return err
		}
		sha1 := fmt.Sprintf("%x",sha1.Sum([]byte(q.Query)))
		if _,ok := all_queries[sha1]; ok && q.State != "active" {
			q.Duration = q.StateChange.Sub(q.QueryStart)
			q.IsDelete = true
			all_queries[sha1] = &q
		}
		if q.State == "active" {
			q.Duration = time.Now().Sub(q.QueryStart)
			all_queries[sha1] = &q
		}
	}
	return nil
}


func main(){
	db_user := utils.GetEnv("PG_USER","postgres")
	db_pass := utils.GetEnv("PG_PASS","")
	db_host := utils.GetEnv("PG_HOST","127.0.0.1")
	db_port := utils.GetEnv("PG_PORT","54322")
	db_name := utils.GetEnv("PG_DBNAME","postgres")


	log.SetFormatter(&log.JSONFormatter{})

	ctx := context.Background()

	db_conn_str := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",db_user,db_pass,db_host,db_port,db_name)
	dbpool, err :=  pgxpool.New(ctx, db_conn_str)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	if err = dbpool.Ping(ctx); err != nil {
		log.Error(err)
		os.Exit(1)
	}
	go func(ctx context.Context, pgpool *pgxpool.Pool) {
		for {
			err = DoQuery(ctx,dbpool)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			time.Sleep(time.Second*1)
		}
	}(ctx,dbpool)

	collector := &QueryCollector{
        metric: prometheus.NewDesc(
            "pg_query",
            "PG Slow query metric",
			[]string{"query"},
            nil,
        ),
    }
    prometheus.MustRegister(collector)

    http.Handle("/metrics", promhttp.Handler())
    log.Info("Starting exporter :8080")
    if err = http.ListenAndServe(":8080", nil); err != nil {
		log.Error(err)
	}
}
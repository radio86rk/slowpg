package querycollector

import (
	"context"
	"crypto/sha1"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"time"
	// log "github.com/sirupsen/logrus"
	"encoding/base64"
	"sync"
)

type Query struct {
	Query       string
	State       string
	Database	string
	QueryID     int64
	QueryStart  time.Time
	StateChange time.Time
	Duration    time.Duration
	IsDelete    bool
}

type QueryCollector struct {
	allQueries map[string]Query
	metric *prometheus.Desc
	mu sync.RWMutex
}

func (qc *QueryCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- qc.metric
}

func (qc *QueryCollector) Collect(ch chan<- prometheus.Metric) {
	tmpQueries := make(map[string]Query)
	qc.mu.RLock()
	for k, v := range qc.allQueries {
		if v.IsDelete {
			tmpQueries[k] = v
		}
	}
	qc.mu.RUnlock()
	for k, v := range tmpQueries {
		metric := prometheus.MustNewConstMetric(qc.metric, prometheus.GaugeValue, float64(v.Duration.Seconds()),v.Database, v.Query )
		ch <- metric
		qc.mu.Lock()
		delete(qc.allQueries, k)
		qc.mu.Unlock()
	}
}

func (qc *QueryCollector) DoQuery(ctx context.Context, pgpool *pgxpool.Pool) error {
	query := `SELECT datname,query,state,query_start,state_change FROM pg_stat_activity WHERE query IS NOT NULL
			  AND state IS NOT NULL AND  query_start IS NOT NULL  AND query NOT LIKE '%pg_stat_activity%'`
	rows, err := pgpool.Query(ctx, query)
	defer rows.Close()
	if err != nil {
		return err
	}

	for rows.Next() {
		q := Query{}
		err = rows.Scan(&q.Database, &q.Query, &q.State, &q.QueryStart, &q.StateChange)
		if err != nil {
			return err
		}
		h := sha1.New()
		h.Write([]byte(q.Query+q.QueryStart.String()))
		hashedQuery := base64.URLEncoding.EncodeToString(h.Sum(nil))
		if q.State == "active"{
			qc.mu.Lock()
			qc.allQueries[hashedQuery] = q
			qc.mu.Unlock()
			continue
		}
		if tmpQuery, ok := qc.allQueries[hashedQuery]; ok {
			tmpQuery.Duration = q.StateChange.Sub(qc.allQueries[hashedQuery].QueryStart)
			tmpQuery.IsDelete = true
			qc.mu.Lock()
			qc.allQueries[hashedQuery] = tmpQuery
			qc.mu.Unlock()
		}

	}
	return nil
}


func New(promMetricName string) *QueryCollector {
	collector := QueryCollector{
		allQueries: make(map[string]Query),
		metric:  prometheus.NewDesc(
            promMetricName,
            "Duration of the query",
			[]string{"db","query"},
            nil,
        ),
	}
	prometheus.MustRegister(&collector)
	return &collector
}

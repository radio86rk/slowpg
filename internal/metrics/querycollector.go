package querycollector

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"time"
	// log "github.com/sirupsen/logrus"
)

type Query struct {
	Query       string
	State       string
	QueryID     int64
	QueryStart  time.Time
	StateChange time.Time
	Duration    time.Duration
	IsDelete    bool
}

type QueryCollector struct {
	allQueries map[string]*Query
	histogram  *prometheus.HistogramVec
}

func (c *QueryCollector) DoQuery(ctx context.Context, pgpool *pgxpool.Pool) error {
	query := "SELECT query,state,query_start,state_change FROM pg_stat_activity WHERE query IS NOT NULL AND state IS NOT NULL AND  query_start IS NOT NULL"
	rows, err := pgpool.Query(ctx, query)
	defer rows.Close()
	if err != nil {
		return err
	}
	for rows.Next() {
		q := Query{}
		err = rows.Scan(&q.Query, &q.State, &q.QueryStart, &q.StateChange)
		if err != nil {
			return err
		}
		hashedQuery := fmt.Sprintf("%x", sha1.Sum([]byte(q.Query+q.QueryStart.String())))
		if q.State == "active" {
			c.allQueries[hashedQuery] = &q
			continue
		}
		if _, ok := c.allQueries[hashedQuery]; ok {
			c.allQueries[hashedQuery].Duration = q.StateChange.Sub(c.allQueries[hashedQuery].QueryStart)
			c.allQueries[hashedQuery].IsDelete = true
		}

	}
	return nil
}

func (c *QueryCollector) CollectMetric() {
	for k, v := range c.allQueries {
		if v.IsDelete {
			//log.Info("Collect query: ",v.Query,"  ",v.Duration.Seconds())
			c.histogram.WithLabelValues(v.Query).Observe(float64(v.Duration.Seconds()))
			delete(c.allQueries, k)
		}
	}
}

func New(promMetricName string) *QueryCollector {
	histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    promMetricName,
		Help:    "Duration of the query.",
		Buckets: []float64{0.1, 0.2, 0.5, 0.75, 1.0, 2.0, 5.0, 10.0},
	}, []string{"query"},
	)
	prometheus.Register(histogramVec)
	return &QueryCollector{
		allQueries: make(map[string]*Query, 100),
		histogram:  histogramVec,
	}
}

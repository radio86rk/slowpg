package pgslowquery

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"pgslowquery/internal/metrics"
	"time"
)

func Run(ctx context.Context,
	pgpool *pgxpool.Pool,
	metricName string,
	scrapms uint,
) error {
	qc := querycollector.New(metricName)
	for {
		select {
		case <-ctx.Done():
			return errors.New("Context canceled")
		default:
			if err := qc.DoQuery(ctx, pgpool); err != nil {
				return err
			}
			time.Sleep(time.Millisecond * time.Duration(scrapms))
			qc.CollectMetric()
		}

	}
	return nil
}

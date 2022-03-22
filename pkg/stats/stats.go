package stats

import (
	"github.com/montanaflynn/stats"
	"go.uber.org/zap"
)

type RequestStats struct {
	lg *zap.Logger

	failCnt    int
	succeedCnt int
	// latency is a cache of all succeeded requests latency in seconds
	// TODO: only cache 1 min of data and then flush them out to cloudwatch metrics
	latency []float64
}

func New(lg *zap.Logger) *RequestStats {
	return &RequestStats{
		lg:      lg,
		latency: make([]float64, 0, 10),
	}
}

func (r *RequestStats) Add(data float64) {
	r.latency = append(r.latency, data)
}
func (r *RequestStats) IncrementFailureCnt() {
	r.failCnt++
}
func (r *RequestStats) IncrementSuccessCnt() {
	r.succeedCnt++
}
func (r *RequestStats) GetFailCnt() int {
	return r.failCnt
}
func (r *RequestStats) GetSuccessCnt() int {
	return r.succeedCnt
}

func (r *RequestStats) Flush() {
	if r.lg == nil {
		return
	}
	r.lg.Info("requests stats",
		zap.Int("total cnt", r.failCnt+r.succeedCnt),
		zap.Int("fail cnt", r.failCnt),
		zap.Float64("fail cnt ratio (percent)", float64(r.failCnt)/float64(r.failCnt+r.succeedCnt) * 100),
		zap.Int("success cnt", r.succeedCnt),
		zap.Float64("success cnt ratio (percent)", float64(r.succeedCnt)/float64(r.failCnt+r.succeedCnt) * 100),
	)
	d := stats.LoadRawData(r.latency)
	min, err := d.Min()
	if err != nil {
		r.lg.Warn("requests stats: fail get min", zap.Float64s("data-set", r.latency), zap.Error(err))
		return
	}
	max, err := d.Max()
	if err != nil {
		r.lg.Warn("requests stats: fail get max", zap.Float64s("data-set", r.latency), zap.Error(err))
		return
	}
	median, err := d.Median()
	if err != nil {
		r.lg.Warn("requests stats: fail get median", zap.Float64s("data-set", r.latency), zap.Error(err))
		return
	}
	avg, err := d.Mean()
	if err != nil {
		r.lg.Warn("requests stats: fail get mean", zap.Float64s("data-set", r.latency), zap.Error(err))
		return
	}
	p50, err := d.Percentile(50)
	if err != nil {
		r.lg.Warn("requests stats: fail get precentile", zap.Float64("percentile", 0.5), zap.Float64s("data-set", r.latency), zap.Error(err))
		return
	}
	p90, err := d.Percentile(90)
	if err != nil {
		r.lg.Warn("requests stats: fail get precentile", zap.Float64("percentile", 0.9), zap.Float64s("data-set", r.latency), zap.Error(err))
		return
	}
	p99, err := d.Percentile(99)
	if err != nil {
		r.lg.Warn("requests stats: fail get precentile", zap.Float64("percentile", 0.99), zap.Float64s("data-set", r.latency), zap.Error(err))
		return
	}

	r.lg.Info("requests stats: latency(s)",
		zap.Float64("min", min),
		zap.Float64("max", max),
		zap.Float64("median", median),
		zap.Float64("mean", avg),
		zap.Float64("p50", p50),
		zap.Float64("p90", p90),
		zap.Float64("p99", p99),
	)
}

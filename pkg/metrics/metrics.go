package metrics

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tel-io/tel/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
)

var (
	valGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type Metrics struct {
	prefix         string
	metricCount    int
	labelCount     int
	seriesCount    int
	metricLength   int
	labelLength    int
	seriesInterval int
	metricInterval int
	constLabels    []string

	metrics []asyncfloat64.Gauge

	labels []attribute.KeyValue

	metricCycle uint
	seriesCycle uint64
	meter       metric.Meter
}

// New creates a set of Prometheus test series that update over time
func New(prefix string, metricCount, labelCount, seriesCount, metricLength, labelLength, seriesInterval, metricInterval int, constLabels []string) (*Metrics, error) {
	labels := make([]attribute.KeyValue, labelCount)
	for idx := 0; idx < labelCount; idx++ {
		labels[idx] = attribute.String(
			fmt.Sprintf("label_key_%s_%v", strings.Repeat("k", labelLength), idx),
			fmt.Sprintf("label_val_%s_%v", strings.Repeat("v", labelLength), idx),
		)
	}

	for _, cLabel := range constLabels {
		split := strings.Split(cLabel, "=")
		if len(split) != 2 {
			return nil, fmt.Errorf("constant label argument must have format labelName=labelValue but got %s", cLabel)
		}
		labels = append(labels, attribute.String(split[0], split[1]))
	}

	return &Metrics{
		prefix:         prefix,
		metricCount:    metricCount,
		labelCount:     labelCount,
		seriesCount:    seriesCount,
		metricLength:   metricLength,
		labelLength:    labelLength,
		seriesInterval: seriesInterval,
		metricInterval: metricInterval,
		constLabels:    constLabels,
		labels:         labels,
		metricCycle:    0,
		seriesCycle:    0,
		meter:          tel.Global().Meter("xxx"),
	}, nil
}

func (m *Metrics) Run(ctx context.Context) error {
	m.metrics = m.registerMetrics()

	seriesTick := time.NewTicker(time.Duration(m.seriesInterval) * time.Second)
	metricTick := time.NewTicker(time.Duration(m.metricInterval) * time.Second)

	go func() {
		for tick := range seriesTick.C {
			tel.FromCtx(ctx).Info("refreshing series cycle", tel.Time("tick", tick))
			atomic.AddUint64(&m.seriesCycle, 1)
			//m.cycleValues()
		}
	}()

	go func() {
		for tick := range metricTick.C {
			fmt.Printf("%v: refreshing metric cycle\n", tick)

			m.metricCycle++
			// One way to do this remove tel
			//unregisterMetrics()
			//m.cycleValues()
		}
	}()

	<-ctx.Done()
	seriesTick.Stop()
	metricTick.Stop()

	return nil
}

func (m *Metrics) cycleValues() {
	for _, metric := range m.metrics {
		for idx := 0; idx < m.seriesCount; idx++ {
			labels := m.seriesLabels(idx)
			metric.Observe(context.Background(), float64(valGenerator.Intn(100)), labels...)
		}
	}
}

func (m *Metrics) registerMetrics() []asyncfloat64.Gauge {
	metrics := make([]asyncfloat64.Gauge, m.metricCount)
	var instr []instrument.Asynchronous

	for idx := 0; idx < m.metricCount; idx++ {
		name := fmt.Sprintf("%s_metric_%s_%v_%v", m.prefix, strings.Repeat("m", m.metricLength), m.metricCycle, idx)
		gauge, err := m.meter.AsyncFloat64().Gauge(name,
			instrument.WithDescription("A tasty metric morsel"))
		noerr("create gauge", err)

		metrics[idx] = gauge
		instr = append(instr, gauge)
	}

	err := m.meter.RegisterCallback(instr, func(ctx context.Context) {
		start := time.Now()

		for _, item := range metrics {
			for idx := 0; idx < m.seriesCount; idx++ {
				labels := m.seriesLabels(idx)
				item.Observe(ctx, float64(valGenerator.Intn(100)), labels...)
			}
		}

		fmt.Println("processing", time.Now().Sub(start).String())
	})

	noerr("RegisterCallback", err)

	return metrics
}

func (m *Metrics) seriesLabels(seriesID int) []attribute.KeyValue {
	return append(m.labels,
		attribute.String("series_id", fmt.Sprintf("%v", seriesID)),
		attribute.String("cycle_id", fmt.Sprintf("%v", atomic.LoadUint64(&m.seriesCycle))),
	)
}

func noerr(msg string, err error) {
	if err != nil {
		tel.Global().Panic(msg, tel.Error(err))
	}
}

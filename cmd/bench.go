package cmd

import (
	"github.com/pkg/errors"
	"github.com/tel-io/compliance/pkg/metrics"
	"github.com/tel-io/tel/v2"
	"github.com/urfave/cli/v2"
)

const (
	metricCount     = "metric-count"
	labelCount      = "label-count"
	seriesCount     = "series-count"
	metricLength    = "metricname-length"
	labelNameLength = "labelname-length"
	valueInterval   = "value-interval"
	labelInterval   = "series-interval"
	metricInterval  = "metric-interval"
	constLabels     = "const-label"
)

type bench struct{}

func newBench() *bench { return &bench{} }

func (c *bench) Command() *cli.Command {
	return &cli.Command{
		Name:         "bench",
		Aliases:      []string{"b"},
		Usage:        "based on avalanche benchmark rool for testing OTEL infrastructure",
		Action:       c.handler(),
		OnUsageError: nil,
		Subcommands:  nil,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    addr,
				Usage:   "OTLP protocol collector address",
				Value:   "0.0.0.0:4317",
				EnvVars: []string{"OTEL_COLLECTOR_GRPC_ADDR"},
			},
			&cli.BoolFlag{
				Name:  insecure,
				Value: true,
				Usage: "insecure grpc connection",
			},
			&cli.IntFlag{
				Name:  metricCount,
				Value: 500,
				Usage: "Number of metrics to serve.",
			},
			&cli.IntFlag{
				Name:  labelCount,
				Value: 10,
				Usage: "Number of labels per-metric.",
			},
			&cli.IntFlag{
				Name:  seriesCount,
				Value: 10,
				Usage: "Number of series per-metric.",
			},
			&cli.IntFlag{
				Name:  metricLength,
				Value: 5,
				Usage: "Modify length of metric names.",
			},
			&cli.IntFlag{
				Name:  labelNameLength,
				Value: 5,
				Usage: "Modify length of label names.",
			},
			&cli.IntFlag{
				Name:  valueInterval,
				Value: 30,
				Usage: "Change series values every {interval} seconds.",
			},
			&cli.IntFlag{
				Name:  labelInterval,
				Value: 60,
				Usage: "Change series_id label values every {interval} seconds.",
			},
			&cli.IntFlag{
				Name:  metricInterval,
				Value: 120,
				Usage: "ToDo: Change __name__ label values every {interval} seconds.",
			},
			&cli.StringSliceFlag{
				Name:  constLabels,
				Usage: "Constant label to add to every metric. Format is labelName=labelValue. Flag can be specified multiple times.",
			},
		},
	}
}

func (c *bench) handler() func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		cfg := tel.GetConfigFromEnv()
		cfg.LogEncode = "console"
		cfg.Namespace = "TEST"
		cfg.Service = "DEMO"
		cfg.Addr = ctx.String(addr)
		cfg.MonitorConfig.Enable = false
		cfg.WithInsecure = ctx.Bool(insecure)

		t, cc := tel.New(ctx.Context, cfg)
		defer cc()

		cxt := tel.WithContext(ctx.Context, t)

		t.Info("collector", tel.String("addr", cfg.Addr))

		m, err := metrics.New(
			ctx.Int(metricCount),
			ctx.Int(labelCount),
			ctx.Int(seriesCount),
			ctx.Int(metricLength),
			ctx.Int(labelNameLength),
			ctx.Int(valueInterval),
			ctx.Int(labelInterval),
			ctx.Int(metricInterval),
			ctx.StringSlice(constLabels),
		)

		if err != nil {
			return errors.WithStack(err)
		}

		stop := make(chan struct{})
		defer close(stop)

		return m.Run(cxt)
	}

}

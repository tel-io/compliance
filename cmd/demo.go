package cmd

import (
	"github.com/tel-io/tel/example/demo/client/v2/pkg/grpctest"
	"github.com/tel-io/tel/example/demo/client/v2/pkg/httptest"
	"github.com/tel-io/tel/example/demo/client/v2/pkg/mgr"
	"github.com/tel-io/tel/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

const (
	threadsNum = "threads"
	httpServer = "http_serer_addr"
	grpcServer = "grpc_server_addr"

	defaultGrpcServer = "0.0.0.0:9500"
	defaultHttpServer = "0.0.0.0:9501"

	sampler = "sampler"
)

type demo struct{}

func newDemo() *demo { return &demo{} }

func (d *demo) Command() *cli.Command {
	return &cli.Command{
		Name:        "demo",
		Aliases:     []string{"d"},
		Description: "send logs with traces + metrics",
		Action:      d.handler(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    addr,
				Usage:   "OTLP protocol collector address",
				Value:   "0.0.0.0:4317",
				EnvVars: []string{"OTEL_COLLECTOR_GRPC_ADDR"},
			},
			&cli.StringFlag{
				Name:  sampler,
				Value: "always",
				Usage: "Tracing sampler: never, always, traceidratio, statustraceidratio. NOTE: traceidratio and statustraceidratio required value separated with ':'. Example: 'traceidratio:10'",
			},
			&cli.BoolFlag{
				Name:  insecure,
				Value: true,
				Usage: "insecure grpc connection",
			},
			&cli.StringFlag{Name: grpcServer, Value: defaultGrpcServer},
			&cli.StringFlag{Name: httpServer, Value: defaultHttpServer},
			&cli.IntFlag{Name: threadsNum, Value: 100, Aliases: []string{"t"}}},
	}
}

func (d *demo) handler() cli.ActionFunc {
	return func(ccx *cli.Context) error {
		cfg := tel.GetConfigFromEnv()
		cfg.LogEncode = "console"
		cfg.Namespace = "TEST"
		cfg.Service = "DEMO"
		cfg.Addr = ccx.String(addr)
		cfg.MonitorConfig.Enable = false
		cfg.WithInsecure = ccx.Bool(insecure)
		cfg.Metrics.EnableRetry = false
		cfg.Traces.EnableSpanTrackLogFields = true
		cfg.Traces.EnableSpanTrackLogMessage = true
		cfg.Traces.Sampler = "always"

		t, cc := tel.New(ccx.Context, cfg)
		defer cc()

		ctx := tel.WithContext(ccx.Context, t)

		t.Info("collector", tel.String("addr", cfg.Addr))

		eg := errgroup.Group{}
		eg.Go(func() error {
			return grpctest.Start(ctx, ccx.String(grpcServer))
		})

		eg.Go(func() error {
			// grpc client
			gClient, err := grpctest.NewClient(&t, ccx.String(grpcServer))
			if err != nil {
				t.Fatal("grpc client", tel.Error(err))
			}

			// http server
			return httptest.New(t, gClient, ccx.String(httpServer)).Start(ctx)
		})

		eg.Go(func() error {
			// http client
			hClt, err := httptest.NewClient(&t, "http://"+ccx.String(httpServer))
			if err != nil {
				t.Fatal("http client", tel.Error(err))
			}

			srv := mgr.New(t, hClt)
			return srv.Start(ctx, 100)
		})

		return eg.Wait()
	}
}

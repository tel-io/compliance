package cmd

import (
	"log"

	"github.com/d7561985/tel/example/demo/client/v2/pkg/grpctest"
	"github.com/d7561985/tel/example/demo/client/v2/pkg/httptest"
	"github.com/d7561985/tel/example/demo/client/v2/pkg/mgr"
	"github.com/d7561985/tel/v2"
	health "github.com/d7561985/tel/v2/monitoring/heallth"
	"github.com/urfave/cli/v2"
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
			&cli.BoolFlag{
				Name:  insecure,
				Value: true,
				Usage: "insecure grpc connection",
			},
		},
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

		t, cc := tel.New(ccx.Context, cfg)
		defer cc()

		ctx := tel.WithContext(ccx.Context, t)
		t.AddHealthChecker(ctx, tel.HealthChecker{Handler: health.NewCompositeChecker()})

		t.Info("collector", tel.String("addr", cfg.Addr))

		// grpc server
		gSrv, err := grpctest.Start()
		if err != nil {
			t.Fatal("grpc server", tel.Error(err))
		}

		// grpc client
		gClient, err := grpctest.NewClient(gSrv)
		if err != nil {
			t.Fatal("grpc client", tel.Error(err))
		}

		// http server
		hAddr, err := httptest.New(gClient).Start()
		if err != nil {
			t.Fatal("http server", tel.Error(err))
		}

		// http client
		hClt, err := httptest.NewClient("http://" + hAddr)
		if err != nil {
			t.Fatal("http client", tel.Error(err))
		}

		srv := mgr.New(t, hClt)
		if err := srv.Start(ctx); err != nil {
			t.Fatal("service", tel.Error(err))
		}

		log.Println("OK")

		return nil
	}
}

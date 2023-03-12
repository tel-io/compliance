package cmd

import (
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/otlplog/otlploggrpc"
	"github.com/tel-io/tel/v2/pkg/logtransform"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.uber.org/zap/zapcore"
)

const (
	addr     = "addr"
	insecure = "insecure"
)

type check struct{}

func newCheck() *check { return &check{} }

func (c *check) Command() *cli.Command {
	return &cli.Command{
		Name:         "check",
		Aliases:      []string{"c"},
		Usage:        "check GRPC target",
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
		},
	}
}

func (c *check) handler() func(*cli.Context) error {
	return func(ccx *cli.Context) error {
		log.Println(ccx.String(addr))

		opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(ccx.String(addr))}
		if ccx.Bool(insecure) {
			opts = append(opts, otlploggrpc.WithInsecure())
		}

		client := otlploggrpc.NewClient(opts...)
		if err := client.Start(ccx.Context); err != nil {
			return errors.WithMessage(err, "start client")
		}

		defer func() {
			_ = client.Stop(ccx.Context)
		}()

		res, _ := resource.New(ccx.Context, resource.WithAttributes(
			// the service name used to display traces in backends
			// key: service.name
			semconv.ServiceNameKey.String("PING"),
			// key: service.namespace
			semconv.ServiceNamespaceKey.String("TEST"),
			// key: service.version
			semconv.ServiceVersionKey.String("TEST"),
			semconv.ServiceInstanceIDKey.String("LOCAL"),
		))

		if err := client.UploadLogs(ccx.Context, logtransform.Trans(res, []logskd.Log{logg()})); err != nil {
			return errors.WithMessagef(err, "send message")
		}

		if err := tr(ccx); err != nil {
			return errors.WithMessage(err, "trace check")
		}

		log.Println("OK")

		return nil
	}

}

func tr(ccx *cli.Context) error {
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(ccx.String(addr))}
	if ccx.Bool(insecure) {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	res, _ := resource.New(ccx.Context, resource.WithAttributes(
		// the service name used to display traces in backends
		// key: service.name
		semconv.ServiceNameKey.String("PING"),
		// key: service.namespace
		semconv.ServiceNamespaceKey.String("TEST"),
		// key: service.version
		semconv.ServiceVersionKey.String("TEST"),
		semconv.ServiceInstanceIDKey.String("LOCAL"),
	))

	client := otlptracegrpc.NewClient(opts...,
	//otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)

	if err := client.Start(ccx.Context); err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		_ = client.Stop(ccx.Context)
	}()

	tp := trace.NewTracerProvider(trace.WithSampler(trace.AlwaysSample()), trace.WithResource(res))
	exp, err := otlptrace.New(ccx.Context, client)
	if err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		_ = tp.Shutdown(ccx.Context)
	}()

	bsp := trace.NewBatchSpanProcessor(exp)
	tp.RegisterSpanProcessor(bsp)

	trx := tp.Tracer("NilExporter")
	_, span := trx.Start(ccx.Context, "XXX")
	span.End()

	return nil
}

func logg() logskd.Log {
	return logskd.NewLog(zapcore.Entry{
		Level:      zapcore.InfoLevel,
		Time:       time.Now(),
		LoggerName: "XXX",
		Message:    "XXX",
	}, attribute.String("HELLO", "WORLD"))
}

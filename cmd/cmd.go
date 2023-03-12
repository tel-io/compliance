package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func Run() {
	ccx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM, os.Interrupt)
		<-c
		cancel()
	}()

	app := &cli.App{
		Name:        "compliance",
		Description: "perform compliance test OTEL infrastructure: logging, metric and tracing via one grpc channel",
		Commands: []*cli.Command{
			newCheck().Command(),
			newDemo().Command(),
			newBench().Command(),
		},
	}

	err := app.RunContext(ccx, os.Args)
	if err != nil {
		log.Fatalln(errors.WithMessage(err, "run application"))
	}
}

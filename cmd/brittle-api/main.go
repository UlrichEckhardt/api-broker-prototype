package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"

	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2" // imports as package cli
)

var (
	logger log15.Logger
)

func main() {
	// setup logger
	logger = log15.New("context", "main")
	logger.SetHandler(log15.StdoutHandler)

	// setup CLI handler
	app := cli.App{
		Name:  "brittle API",
		Usage: "This serves a nonreliable API",
		Flags: []cli.Flag{},
		Commands: []*cli.Command{
			{
				Name:      "serve",
				Usage:     "Serve API.",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					&cli.Float64Flag{
						Name:  "api-failure-rate",
						Value: 0.0,
						Usage: "Fraction of requests that fail.",
					},
					&cli.Float64Flag{
						Name:  "api-silent-failure-rate",
						Value: 0.0,
						Usage: "Fraction of requests that don't produce any response.",
					},
					&cli.Float64Flag{
						Name:  "api-min-latency",
						Value: 0.0,
						Usage: "Minimal handling delay.",
					},
					&cli.Float64Flag{
						Name:  "api-max-latency",
						Value: 0.0,
						Usage: "Maximal handling delay.",
					},
				},
				Action: func(c *cli.Context) error {
					args := c.Args()
					if args.Len() != 1 {
						return errors.New("exactly one argument expected")
					}
					addr := args.First()

					failureRate := c.Float64("api-failure-rate")
					silentFailureRate := c.Float64("api-silent-failure-rate")
					minLatency := c.Float64("api-min-latency")
					maxLatency := c.Float64("api-max-latency")

					return serveMain(c.Context, failureRate, silentFailureRate, minLatency, maxLatency, addr)
				},
			},
		},
	}

	// setup a context for coordinated shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// run the actual commandline application
	if err := app.RunContext(ctx, os.Args); err != nil {
		logger.Debug("command exited with error", "error", err)
		return
	}
}

func serveMain(ctx context.Context, failureRate float64, silentFailureRate float64, minLatency float64, maxLatency float64, addr string) error {
	server := &http.Server{Addr: addr, Handler: nil}
	go func() {
		<-ctx.Done()
		logger.Info("shutting down")
		server.Shutdown(context.Background())
	}()
	logger.Info("listening for incoming connections", "addr", addr)
	return server.ListenAndServe()
}

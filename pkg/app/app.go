package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/fatih/color"
	"github.com/readly/eb-listener/pkg/eb"
	"github.com/readly/eb-listener/pkg/listen"
	"github.com/rodaine/table"
	"github.com/rs/xid"
	"github.com/urfave/cli/v2"
)

var RunID xid.ID

func init() {
	RunID = xid.New()
}

var CLI = &cli.App{
	Commands: []*cli.Command{
		{
			Name:  "list",
			Usage: "List available buses",
			Action: func(*cli.Context) error {
				cfg, err := config.LoadDefaultConfig(context.TODO())
				if err != nil {
					return fmt.Errorf("Failed to configure aws %w", err)
				}

				eb := eventbridge.NewFromConfig(cfg)
				output, err := eb.ListEventBuses(context.TODO(), nil)
				if err != nil {
					return fmt.Errorf("Failed to list eventbuses %w", err)
				}

				headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
				columnFmt := color.New(color.FgYellow).SprintfFunc()

				tbl := table.New("Name", "ARN").WithWriter(os.Stdout)
				tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

				for i := range output.EventBuses {
					b := output.EventBuses[i]
					tbl.AddRow(*b.Name, *b.Arn)
				}
				tbl.Print()
				return nil
			},
		},
		{
			Name:  "listen",
			Usage: "Start to listen to all messages on a AWS eventbridge bus",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "bus",
					Usage:    "Name of the eventbridge bus",
					Required: true,
				},
				&cli.BoolFlag{
					Name:  "verbose",
					Usage: "Get more verbose output",
					Value: false,
				},
			},
			Action: func(cCtx *cli.Context) error {
				slog.SetDefault(slog.Default().With("id", RunID.String()))

				if cCtx.Bool("verbose") {
					slog.SetLogLoggerLevel(slog.LevelDebug)
				}

				// create a SQS queue
				// Create a rule on bus with SQS as target
				// Start listening on queue
				cfg, err := config.LoadDefaultConfig(context.TODO())
				if err != nil {
					return fmt.Errorf("Failed to configure aws %w", err)
				}

				// Initiate bus
				slog.Debug("initiating eventbus")
				bus, err := eb.NewBus(cfg, RunID, cCtx.String("bus"))
				if err != nil {
					return err
				}
				defer bus.Cleanup(context.TODO())

				// Initiate listener
				slog.Debug("initiating sqs")
				s, err := listen.NewSQS(cfg, RunID)
				if err != nil {
					return fmt.Errorf("failed to start listener %w", err)
				}

				// Start to listen for messages
				s.Listen(context.Background())

				defer s.Shutdown(context.TODO())

				// Attach listener to bus
				err = bus.AttachSQS(context.Background(), s)
				if err != nil {
					return err
				}

				osSignal := make(chan os.Signal, 1)
				signal.Notify(osSignal, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

				for {
					select {
					case sig := <-osSignal:
						slog.Info("received signal, shutdown initiated", "signal", sig.String())
						return nil
					}
				}
			},
		},
	},
}

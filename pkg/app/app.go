package app

import (
	"context"
	"errors"
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
	snssource "github.com/readly/eb-listener/pkg/sns"
	"github.com/rodaine/table"
	"github.com/rs/xid"
	"github.com/urfave/cli/v2"
)

var RunID xid.ID

type messageSource interface {
	AttachSQS(context.Context, *listen.SQS) error
	Cleanup(context.Context) error
}

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
			Usage: "Listen to messages on an AWS EventBridge bus or SNS topic",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "bus",
					Usage: "Name of the EventBridge bus",
				},
				&cli.StringFlag{
					Name:  "topic",
					Usage: "Name or ARN of the SNS topic",
				},
				&cli.BoolFlag{
					Name:  "sns-envelope",
					Usage: "Keep the SNS notification envelope instead of using raw message delivery",
				},
				&cli.BoolFlag{
					Name:  "fifo",
					Usage: "Create a FIFO SQS queue for EventBridge",
					Value: false,
				},
				&cli.BoolFlag{
					Name:  "only-detail",
					Usage: "Only output the EventBridge event detail field",
					Value: false,
				},
				&cli.BoolFlag{
					Name:  "verbose",
					Usage: "Pretty-print events and enable debug logs",
					Value: false,
				},
			},
			Action: func(cCtx *cli.Context) error {
				slog.SetDefault(slog.Default().With("run-id", RunID.String()))
				busName := cCtx.String("bus")
				topicName := cCtx.String("topic")
				if (busName == "") == (topicName == "") {
					return errors.New("exactly one of --bus or --topic must be specified")
				}
				if topicName != "" && cCtx.Bool("only-detail") {
					return errors.New("--only-detail is only supported with --bus")
				}
				if topicName != "" && cCtx.Bool("fifo") {
					return errors.New("--fifo is selected automatically for FIFO SNS topics")
				}
				if busName != "" && cCtx.Bool("sns-envelope") {
					return errors.New("--sns-envelope is only supported with --topic")
				}

				if cCtx.Bool("verbose") {
					slog.SetLogLoggerLevel(slog.LevelDebug)
				}

				cfg, err := config.LoadDefaultConfig(context.TODO())
				if err != nil {
					return fmt.Errorf("Failed to configure aws %w", err)
				}

				var source messageSource
				fifo := cCtx.Bool("fifo")
				if topicName != "" {
					topic, err := snssource.NewTopic(context.TODO(), cfg, topicName, !cCtx.Bool("sns-envelope"))
					if err != nil {
						return err
					}
					source = topic
					fifo = topic.IsFIFO()
				} else {
					bus, err := eb.NewBus(cfg, RunID, busName)
					if err != nil {
						return err
					}
					source = bus
				}

				// Initiate listener
				slog.Debug("initiating sqs")
				s, err := listen.NewSQS(cfg, RunID, fifo, cCtx.Bool("only-detail"), cCtx.Bool("verbose"))
				if err != nil {
					return fmt.Errorf("failed to start listener %w", err)
				}

				s.Listen(context.Background())
				defer func() {
					if err := s.Shutdown(context.TODO()); err != nil {
						slog.Error("failed to shut down SQS listener", "error", err)
					}
				}()
				defer func() {
					if err := source.Cleanup(context.TODO()); err != nil {
						slog.Error("failed to clean up message source", "error", err)
					}
				}()

				err = source.AttachSQS(context.Background(), s)
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

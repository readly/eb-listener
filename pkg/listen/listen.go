package listen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/rs/xid"
)

const (
	SQSQueuePrefix = "eb-listener"
)

type SQS struct {
	runID      xid.ID
	config     aws.Config
	client     *sqs.Client
	isFIFO     bool
	msgChan    chan receivedEvent
	queueName  string
	QueueURL   string
	QueueARN   string
	ctxCancel  context.CancelFunc
	wg         *sync.WaitGroup
	log        *slog.Logger
	onlyDetail bool
	verbose    bool
	printed    bool
}

type receivedEvent struct {
	event Event
	raw   json.RawMessage
}

func (s *SQS) IsFIFO() bool {
	return s.isFIFO
}

func (s *SQS) cleanup(ctx context.Context) error {
	if s.QueueURL != "" {
		_, err := s.client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
			QueueUrl: &s.QueueURL,
		})
		if err != nil {
			return fmt.Errorf("failed to delete sqs queue %w", err)
		}
		s.log.Info("deleting sqs queue")
	}
	return nil
}

func (s *SQS) Listen(ctx context.Context) {
	ctx, s.ctxCancel = context.WithCancel(ctx)
	s.log.Info("start listening for messages on SQS queue")
	s.wg.Add(2)
	go s.pollMessages(ctx)
	go s.printMessages(ctx)
}

func (s *SQS) Shutdown(ctx context.Context) error {
	s.ctxCancel()
	s.wg.Wait()

	err := s.cleanup(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQS) printMessages(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case msg, ok := <-s.msgChan:
			if !ok {
				s.log.Debug("msg channel closed")
				return
			}
			if err := s.printMessage(os.Stdout, msg); err != nil {
				s.log.Error("failed to print event", "error", err)
			}
		case <-ctx.Done():
			s.log.Debug("stopping message printing for sqs queue")
			return
		}
	}
}

func (s *SQS) printMessage(w io.Writer, msg receivedEvent) error {
	if s.verbose {
		if s.printed {
			if _, err := fmt.Fprintln(w, "---"); err != nil {
				return fmt.Errorf("failed to write message divider: %w", err)
			}
		}
		s.printed = true

		payload := msg.raw
		if s.onlyDetail {
			payload = msg.event.Detail
		}
		return printPrettyJSON(w, payload)
	}

	if s.onlyDetail {
		s.log.Info("received event", "id", msg.event.ID, "detail-type", msg.event.DetailType, "detail", string(msg.event.Detail))
		return nil
	}

	s.log.Info("received event", "event", string(msg.raw))
	return nil
}

func printPrettyJSON(w io.Writer, payload json.RawMessage) error {
	var out bytes.Buffer
	if err := json.Indent(&out, payload, "", "  "); err != nil {
		out.Reset()
		out.Write(payload)
	}

	if _, err := fmt.Fprintln(w, out.String()); err != nil {
		return fmt.Errorf("failed to write json: %w", err)
	}

	return nil
}

func (s *SQS) pollMessages(ctx context.Context) {
	defer s.wg.Done()
	for {
		result, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(s.QueueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     1,
		})
		if err != nil {
			if ctx.Err() != nil {
				s.log.Debug("stopping message polling on sqs queue")
				return
			}
			s.log.Error("failed to receive messages", "error", err)
			continue
		}

		deleteInput := sqs.DeleteMessageBatchInput{
			QueueUrl: &s.QueueURL,
			Entries:  make([]types.DeleteMessageBatchRequestEntry, 0, len(result.Messages)),
		}
		for i := range result.Messages {
			var event Event
			raw := json.RawMessage(*result.Messages[i].Body)
			_ = json.Unmarshal(raw, &event)
			select {
			case s.msgChan <- receivedEvent{event: event, raw: raw}:
			case <-ctx.Done():
				return
			}
			deleteInput.Entries = append(deleteInput.Entries, types.DeleteMessageBatchRequestEntry{
				Id:            result.Messages[i].MessageId,
				ReceiptHandle: result.Messages[i].ReceiptHandle,
			})
		}

		if len(deleteInput.Entries) > 0 {
			if _, err = s.client.DeleteMessageBatch(ctx, &deleteInput); err != nil {
				s.log.Error("failed to delete messages", "error", err)
			}
		}
	}
}

func (s *SQS) AllowMessagesFrom(ctx context.Context, principal string, sourceARN string) error {
	_, err := s.client.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{
		QueueUrl: &s.QueueURL,
		Attributes: map[string]string{
			"Policy": NewIamSqsPolicy(fmt.Sprintf("%s-%s", SQSQueuePrefix, s.runID), s.QueueARN, principal, sourceARN),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to attach policy to queue %w", err)
	}
	return nil
}

func NewSQS(cfg aws.Config, id xid.ID, fifo bool, onlyDetail bool, verbose bool) (*SQS, error) {
	queueName := fmt.Sprintf("%s-%s", SQSQueuePrefix, id.String())
	attributes := map[string]string{}
	if fifo {
		queueName = fmt.Sprintf("%s.fifo", queueName)
		attributes["FifoQueue"] = "true"
		attributes["ContentBasedDeduplication"] = "true"
	}

	s := &SQS{
		config:     cfg,
		queueName:  queueName,
		isFIFO:     fifo,
		msgChan:    make(chan receivedEvent),
		wg:         new(sync.WaitGroup),
		runID:      id,
		log:        slog.Default(),
		onlyDetail: onlyDetail,
		verbose:    verbose,
	}
	s.client = sqs.NewFromConfig(cfg)

	resp, err := s.client.CreateQueue(context.TODO(), &sqs.CreateQueueInput{
		QueueName:  &s.queueName,
		Attributes: attributes,
		Tags: map[string]string{
			"autogenerated": "true",
			"user":          os.Getenv("USER"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sqs queue %w", err)
	}

	s.QueueURL = *resp.QueueUrl

	resp2, err := s.client.GetQueueAttributes(context.TODO(), &sqs.GetQueueAttributesInput{
		QueueUrl: &s.QueueURL,
		AttributeNames: []types.QueueAttributeName{
			"QueueArn",
		},
	})
	if err != nil {
		errClean := s.cleanup(context.TODO())
		if errClean != nil {
			s.log.Error("failed to clean up SQS queue", "error", errClean)
		}
		return nil, fmt.Errorf("failed to get queue arn %w", err)
	}

	s.QueueARN = resp2.Attributes["QueueArn"]

	s.log = slog.Default().With("url", s.QueueURL)

	s.log.Info("created SQS queue", "arn", s.QueueARN)

	return s, nil
}

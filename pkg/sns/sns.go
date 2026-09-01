package sns

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/readly/eb-listener/pkg/listen"
)

type client interface {
	ListTopics(context.Context, *awssns.ListTopicsInput, ...func(*awssns.Options)) (*awssns.ListTopicsOutput, error)
	Subscribe(context.Context, *awssns.SubscribeInput, ...func(*awssns.Options)) (*awssns.SubscribeOutput, error)
	Unsubscribe(context.Context, *awssns.UnsubscribeInput, ...func(*awssns.Options)) (*awssns.UnsubscribeOutput, error)
}

type Topic struct {
	client          client
	arn             string
	rawDelivery     bool
	subscriptionARN string
	log             *slog.Logger
}

func NewTopic(ctx context.Context, cfg aws.Config, topic string, rawDelivery bool) (*Topic, error) {
	client := awssns.NewFromConfig(cfg)
	topicARN, err := resolveTopicARN(ctx, client, cfg.Region, topic)
	if err != nil {
		return nil, err
	}

	return &Topic{
		client:      client,
		arn:         topicARN,
		rawDelivery: rawDelivery,
		log:         slog.Default().With("topic", topicARN),
	}, nil
}

func resolveTopicARN(ctx context.Context, client awssns.ListTopicsAPIClient, region string, topic string) (string, error) {
	if arn.IsARN(topic) {
		parsed, err := arn.Parse(topic)
		if err != nil {
			return "", fmt.Errorf("failed to parse topic ARN: %w", err)
		}
		if parsed.Service != "sns" || parsed.Region != region || parsed.Resource == "" {
			return "", fmt.Errorf("topic ARN must identify an SNS topic in region %s", region)
		}
		return topic, nil
	}

	paginator := awssns.NewListTopicsPaginator(client, &awssns.ListTopicsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to list sns topics: %w", err)
		}
		for _, candidate := range page.Topics {
			if candidate.TopicArn != nil && strings.HasSuffix(*candidate.TopicArn, ":"+topic) {
				return *candidate.TopicArn, nil
			}
		}
	}

	return "", fmt.Errorf("sns topic %q not found", topic)
}

func (t *Topic) IsFIFO() bool {
	return strings.HasSuffix(t.arn, ".fifo")
}

func (t *Topic) AttachSQS(ctx context.Context, queue *listen.SQS) error {
	if err := queue.AllowMessagesFrom(ctx, "sns.amazonaws.com", t.arn); err != nil {
		return err
	}
	return t.subscribe(ctx, queue.QueueARN)
}

func (t *Topic) subscribe(ctx context.Context, queueARN string) error {
	output, err := t.client.Subscribe(ctx, &awssns.SubscribeInput{
		TopicArn:              aws.String(t.arn),
		Protocol:              aws.String("sqs"),
		Endpoint:              aws.String(queueARN),
		ReturnSubscriptionArn: true,
		Attributes: map[string]string{
			"RawMessageDelivery": fmt.Sprintf("%t", t.rawDelivery),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe sqs queue to sns topic: %w", err)
	}
	if output.SubscriptionArn == nil || *output.SubscriptionArn == "pending confirmation" {
		return errors.New("sns subscription was not confirmed")
	}

	t.subscriptionARN = *output.SubscriptionArn
	t.log.Info("subscribed SQS queue to SNS topic", "queue", queueARN)
	return nil
}

func (t *Topic) Cleanup(ctx context.Context) error {
	if t.subscriptionARN == "" {
		return nil
	}

	if _, err := t.client.Unsubscribe(ctx, &awssns.UnsubscribeInput{
		SubscriptionArn: aws.String(t.subscriptionARN),
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe from sns topic: %w", err)
	}

	t.log.Info("unsubscribed from SNS topic")
	return nil
}

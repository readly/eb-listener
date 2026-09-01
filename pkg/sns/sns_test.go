package sns

import (
	"context"
	"log/slog"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type fakeClient struct {
	topics           []types.Topic
	subscribeInput   *awssns.SubscribeInput
	unsubscribeInput *awssns.UnsubscribeInput
}

func (f *fakeClient) ListTopics(context.Context, *awssns.ListTopicsInput, ...func(*awssns.Options)) (*awssns.ListTopicsOutput, error) {
	return &awssns.ListTopicsOutput{Topics: f.topics}, nil
}

func (f *fakeClient) Subscribe(_ context.Context, input *awssns.SubscribeInput, _ ...func(*awssns.Options)) (*awssns.SubscribeOutput, error) {
	f.subscribeInput = input
	return &awssns.SubscribeOutput{SubscriptionArn: aws.String("arn:aws:sns:eu-west-1:123456789012:topic:subscription-id")}, nil
}

func (f *fakeClient) Unsubscribe(_ context.Context, input *awssns.UnsubscribeInput, _ ...func(*awssns.Options)) (*awssns.UnsubscribeOutput, error) {
	f.unsubscribeInput = input
	return &awssns.UnsubscribeOutput{}, nil
}

func TestResolveTopicARNByName(t *testing.T) {
	client := &fakeClient{topics: []types.Topic{
		{TopicArn: aws.String("arn:aws:sns:eu-west-1:123456789012:other")},
		{TopicArn: aws.String("arn:aws:sns:eu-west-1:123456789012:orders")},
	}}

	got, err := resolveTopicARN(context.Background(), client, "eu-west-1", "orders")
	if err != nil {
		t.Fatalf("resolveTopicARN() error = %v", err)
	}
	if want := "arn:aws:sns:eu-west-1:123456789012:orders"; got != want {
		t.Fatalf("resolveTopicARN() = %q, want %q", got, want)
	}
}

func TestResolveTopicARNRejectsAnotherRegion(t *testing.T) {
	_, err := resolveTopicARN(context.Background(), &fakeClient{}, "eu-west-1", "arn:aws:sns:us-east-1:123456789012:orders")
	if err == nil {
		t.Fatal("resolveTopicARN() error = nil, want region error")
	}
}

func TestSubscribeAndCleanup(t *testing.T) {
	client := &fakeClient{}
	topic := &Topic{
		client:      client,
		arn:         "arn:aws:sns:eu-west-1:123456789012:orders.fifo",
		rawDelivery: true,
		log:         slog.Default(),
	}

	if !topic.IsFIFO() {
		t.Fatal("IsFIFO() = false, want true")
	}
	if err := topic.subscribe(context.Background(), "arn:aws:sqs:eu-west-1:123456789012:queue.fifo"); err != nil {
		t.Fatalf("subscribe() error = %v", err)
	}
	if got, want := client.subscribeInput.Attributes["RawMessageDelivery"], "true"; got != want {
		t.Fatalf("RawMessageDelivery = %q, want %q", got, want)
	}
	if got, want := aws.ToString(client.subscribeInput.Endpoint), "arn:aws:sqs:eu-west-1:123456789012:queue.fifo"; got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}

	if err := topic.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if client.unsubscribeInput == nil {
		t.Fatal("Cleanup() did not unsubscribe")
	}
}

func TestResolveTopicARNReturnsNotFound(t *testing.T) {
	_, err := resolveTopicARN(context.Background(), &fakeClient{}, "eu-west-1", "missing")
	if err == nil {
		t.Fatal("resolveTopicARN() error = nil, want not found error")
	}
}

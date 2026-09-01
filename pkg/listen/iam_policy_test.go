package listen

import (
	"encoding/json"
	"testing"
)

func TestNewIamSqsPolicyScopesSource(t *testing.T) {
	var policy IamPolicy
	if err := json.Unmarshal([]byte(NewIamSqsPolicy(
		"listener-id",
		"arn:aws:sqs:eu-west-1:123456789012:queue",
		"sns.amazonaws.com",
		"arn:aws:sns:eu-west-1:123456789012:topic",
	)), &policy); err != nil {
		t.Fatalf("failed to unmarshal policy: %v", err)
	}

	statement := policy.Statement[0]
	if got, want := statement.Principal["Service"], "sns.amazonaws.com"; got != want {
		t.Fatalf("principal = %q, want %q", got, want)
	}
	if got, want := statement.Resource, "arn:aws:sqs:eu-west-1:123456789012:queue"; got != want {
		t.Fatalf("resource = %q, want %q", got, want)
	}
	if got, want := statement.Condition["ArnEquals"]["aws:SourceArn"], "arn:aws:sns:eu-west-1:123456789012:topic"; got != want {
		t.Fatalf("source ARN = %q, want %q", got, want)
	}
}

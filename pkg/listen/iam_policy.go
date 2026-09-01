package listen

import (
	"encoding/json"
)

type IamStatement struct {
	Sid       string
	Effect    string
	Principal map[string]string
	Action    string
	Resource  string
	Condition map[string]map[string]string
}

type IamPolicy struct {
	Version   string
	Id        string
	Statement []IamStatement
}

func NewIamSqsPolicy(id string, queueARN string, principal string, sourceARN string) string {
	policy := IamPolicy{
		Version: "2012-10-17",
		Id:      id,
		Statement: []IamStatement{
			{
				Sid:    id,
				Effect: "Allow",
				Principal: map[string]string{
					"Service": principal,
				},
				Action:   "sqs:SendMessage",
				Resource: queueARN,
				Condition: map[string]map[string]string{
					"ArnEquals": {
						"aws:SourceArn": sourceARN,
					},
				},
			},
		},
	}
	json, _ := json.Marshal(policy)
	return string(json)
}

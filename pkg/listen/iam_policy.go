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
}

type IamPolicy struct {
	Version   string
	Id        string
	Statement []IamStatement
}

func NewIamSqsPolicy(id string, queueArn string) string {
	policy := IamPolicy{
		Version: "2012-10-17",
		Id:      id,
		Statement: []IamStatement{
			{
				Sid:    id,
				Effect: "Allow",
				Principal: map[string]string{
					"Service": "events.amazonaws.com",
				},
				Action:   "sqs:SendMessage",
				Resource: queueArn,
			},
		},
	}
	json, _ := json.Marshal(policy)
	return string(json)
}

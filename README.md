# AWS EventBridge and SNS Listener (eb-listener)

Listens for messages on an AWS EventBridge bus or SNS topic and outputs them to
the terminal.

## Examples

To list all event buses in `us-west-1`.

`$ AWS_REGION=us-west-1 eb-listener list`

Use a different AWS credential profile.

`$ AWS_PROFILE=secret eb-listener listen --bus pinkbus`

By default, received messages are output as log rows containing the full
EventBridge event.

Pretty-print the full EventBridge event JSON.

`$ AWS_PROFILE=secret eb-listener listen --bus pinkbus --verbose`

Only output the EventBridge event detail field as a log row.

`$ AWS_PROFILE=secret eb-listener listen --bus pinkbus --only-detail`

Pretty-print only the EventBridge event detail JSON.

`$ AWS_PROFILE=secret eb-listener listen --bus pinkbus --only-detail --verbose`

Create a FIFO queue for listening.

`$ AWS_PROFILE=secret eb-listener listen --bus pinkbus --fifo`

Listen to an SNS topic by name or ARN. Topic names are resolved in the configured
AWS account and region. SNS messages use raw message delivery by default.

`$ AWS_PROFILE=secret eb-listener listen --topic orders`

Keep the SNS notification envelope instead of receiving the published message
directly.

`$ AWS_PROFILE=secret eb-listener listen --topic orders --sns-envelope`

FIFO SNS topics automatically use a FIFO SQS queue.

## AWS Access

Your AWS credentials will need to have access to create, update and delete SQS
queues. EventBridge listening also requires access to create and delete rules
and targets. SNS listening requires `sns:ListTopics` when using a topic name,
plus `sns:Subscribe` and `sns:Unsubscribe`. Using a topic ARN does not require
`sns:ListTopics`.

## How it works

`eb-listener` creates an SQS queue. For EventBridge it adds a rule that catches
all events and targets the queue. For SNS it subscribes the queue to the topic.
The queue policy only permits messages from the generated EventBridge rule or
selected SNS topic.

`eb-listener` then starts to poll the SQS queue for new messages.

When you are done listening hit CTRL-C and `eb-listener` will clean up the
EventBridge rule and target or SNS subscription, followed by the SQS queue.

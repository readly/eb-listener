# Eventbridge Listener (eb-listener)

Listens for messages on a AWS EventBridge bus and outputs them to the terminal.

## Examples

To list all event buses in `us-west-1`.

`$ AWS_REGION=us-west-1 eb-listener list`

Use a different AWS credential profile.

`$ AWS_PROFILE=secret eb-listener listen --bus pinkbus`

## AWS Access

Your AWS credentials will need to have access to create, update and delete SQS
queues. It will also need access to the EventBridge bus to be able to create
rules and targets.

## How it works

`eb-listener` creates a SQS queue. Then it adds a rule to catch all events on a
EventBridge bus and attaches a target to the SQS queue.

`eb-listener` then starts to poll the SQS queue for new messages.

When you are done listening hit CTRL-C and `eb-listener` will clean up the
EventBridge Rule and Target and also the SQS queue.

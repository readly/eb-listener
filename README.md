# Eventbridge Listener (eb-listener)

Listens for messages on a AWS EventBridge bus and outputs them to the terminal.

## Examples

To list all event buses in `us-west-1`.

`$ AWS_REGION=us-west-1 eb-listener list`

Use a different AWS credential profile.

`$ AWS_PROFILE=secret eb-listener listen --bus pinkbus`

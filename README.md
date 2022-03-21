# RSMQ

**Guaranteed delivery of a message to exactly one recipient within a message visibility timeout.**


This is an implementation of https://github.com/smrchy/rsmq but for the go language.

It is tested for compatibility against the smrchy/rsmq implementation in nodejs.

# Implemented

- CreateQueue
- GetQueueAttributes
- SendMessage
- RecieveMessage
- DeleteQueue
- DeleteMessage
- ChangeMessageVisibility
- Close connection
- ListQueues
- PopMessage

## TODO

- Support REALTIME
- Implement SetQueueAttributes

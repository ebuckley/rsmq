# RSMQ

**Guaranteed delivery of a message to exactly one recipient within a message visibility timeout.**


This is an implementation of https://github.com/smrchy/rsmq but for the go language.

It is tested for compatibility against the smrchy/rsmq implementation in nodejs.

## getting started
```go
// see cmd/simple/main.go for a full example
package main

func main() {
    queue, _ := q.New(ctx, q.Options{})
    
    qname := "SimpleGOTEST"
    _ = queue.CreateQueue(ctx, q.CreateQueueRequestOptions{QName: qname})
    
    attributes, _ := queue.GetQueueAttributes(ctx, q.GetQueueAttributesOptions{QName: qname})

    log.Println("Attributes found!:\n", attributes)
    
    uid, _ := queue.SendMessage(ctx, q.SendMessageRequestOptions{
        QName:   qname,
        Message: "HELLO WORLD!",
    })
    log.Println("Sent message with uid: ", uid)
    
    message, _ := queue.ReceiveMessage(ctx, q.ReceiveMessageOptions{QName: qname})
    
    log.Println("Got message off the queue", message)
}
```

## The worker framework

Inspired by the excellent http package in go and [rsmq-worker](https://github.com/mpneuried/rsmq-worker). 
This allows you to easily create an async worker that consumes messages form the Queue. Importantly it provides a hook for deadline passed.
Deadline so that you can handle a situation where a message is received twice.

```go
// worker/worker.go
type Handler interface {
	Message(ctx context.Context, msg *q.Message) (bool, error)
	Error(ctx context.Context, err error, msg *q.Message) error
	DeadlinePassed(ctx context.Context, msg *q.Message) error
}
```


# Implemented

Progress towards API compatibility with `smrchy/rsmq`.

- Worker framework
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

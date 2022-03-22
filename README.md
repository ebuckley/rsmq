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
# Why RSMQ?

In `$current_year` there are a whole suite of possible tools you can use for queueing, why choose this one? You might be asking. Why not kafka? Why not SQS?
These behemoth tools fit a place, the RSMQ protocol is a small and beautiful alternative.

**Simple**
An implementation of RSMQ comes in at under 500 lines of code. You can write it in a weekend. A small matter of story points and coding. Beyond a small number of lines of code, it is simple in API surface. Your queue processing logic becomes front matter for your implementation, not being ground down in to implementation detail and framework gymnastics.

**Solid base**
This QUEUE is built on redis, we offload the guarantees of the data layer to redis. It makes the queue an obvious choice if your data layer is already built on Redis.

**Exactly once delivery***
Messages are delivered exactly once within a visibility timeout period.

**Decouple services**
Implementations of the RSMQ are so simple that you can implement them in any language in a couple of days effort. There are mature implementations for most languages. You should pick one of the following.

Other implementations:

- [nodejs](https://github.com/smrchy/rsmq)
- [php](https://github.com/eislambey/php-rsmq)
- [python](https://github.com/eislambey/php-rsmq)
- [rust](https://github.com/eislambey/php-rsmq)
- [c#](https://github.com/tontonrally/rsmqCsharp)
- [java](https://github.com/igr/jrsmq)


# How it works?

I wrote a short tour of the codebase [here](https://dev.to/ebuckley/rsmq-for-golang-2ej5).

The implementation weighs in at less than 500 SLOC, so this is something you can read end to end in a couple hours if you please.

# Progress report

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
- RSMQ rest API
- RSMQ ui

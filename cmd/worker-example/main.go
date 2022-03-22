package main

import (
	"context"
	"github.com/ebuckley/rsmq/q"
	"github.com/ebuckley/rsmq/worker"
	"github.com/go-redis/redis/v8"
	"log"
	"os"
)

const qName = "main"

var ns = "worker-example"

type myWorker struct{}

func (m myWorker) Message(ctx context.Context, msg *q.Message) (bool, error) {
	log.Println("myWorker got message:\n", msg)
	return true, nil
}

func (m myWorker) Error(ctx context.Context, err error, msg *q.Message) error {
	log.Println("Got error message:", err, "\n", msg)
	return nil
}

func (m myWorker) DeadlinePassed(ctx context.Context, msg *q.Message) error {
	log.Println("Deadline passed, undo this message processing:\n", msg)
	return nil
}

func main() {
	ctx := context.Background()
	url := os.Getenv("REDIS_URL")
	if len(url) == 0 {
		url = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalln("parseURL: %w", err)
	}

	mq, err := q.New(ctx, q.Options{
		Client:    redis.NewClient(opts),
		NameSpace: &ns,
	})
	err = mq.CreateQueue(ctx, q.CreateQueueRequestOptions{QName: qName})
	if err != nil {
		log.Println("Warning: Q create issue:", err)
	}
	if len(os.Args) < 2 {
		log.Println(`
worker-example is an example of creating a persistent worker and
listen for messages USAGE: worker-example listen
send a message      USAGE: worker-example send "hello world!"
  `)
	}
	if os.Args[1] == "send" {
		_, err = mq.SendMessage(ctx, q.SendMessageRequestOptions{
			QName:   qName,
			Message: os.Args[2],
		})
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	if os.Args[1] == "listen" {
		hand := myWorker{}
		worker.New(ctx, mq, qName, hand).Start()
	}
}

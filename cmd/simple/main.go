package main

import (
	"context"
	"github.com/ebuckley/rsmq/q"
	"log"
)

func main() {
	ctx := context.Background()
	queue, err := q.New(ctx, q.Options{})
	if err != nil {
		log.Fatalln(err)
	}
	qname := "SimpleGOTEST"
	err = queue.CreateQueue(ctx, q.CreateQueueRequestOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	attributes, err := queue.GetQueueAttributes(ctx, q.GetQueueAttributesOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Attributes found!:\n", attributes)

	uid, err := queue.SendMessage(ctx, q.SendMessageRequestOptions{
		QName:   qname,
		Delay:   0,
		Message: "HELLO WORLD!",
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Send message with uid: ", uid)

	message, err := queue.ReceiveMessage(ctx, q.ReceiveMessageOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Got result", message)
}

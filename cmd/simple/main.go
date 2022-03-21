package main

import (
	"context"
	"github.com/ebuckley/rsmq"
	"log"
)

func main() {
	ctx := context.Background()
	q, err := rsmq.New(ctx, rsmq.Options{})
	if err != nil {
		log.Fatalln(err)
	}
	qname := "SimpleGOTEST"
	err = q.CreateQueue(ctx, rsmq.CreateQueueRequestOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	attributes, err := q.GetQueueAttributes(ctx, rsmq.GetQueueAttributesOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Attributes found!:\n", attributes)

	uid, err := q.SendMessage(ctx, rsmq.SendMessageRequestOptions{
		QName:   qname,
		Delay:   0,
		Message: "HELLO WORLD!",
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Send message with uid: ", uid)

	message, err := q.ReceiveMessage(ctx, rsmq.ReceiveMessageOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Got result", message)
}

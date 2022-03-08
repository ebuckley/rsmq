package main

import (
	"context"
	"log"
	"rsmq"
)

func main() {
	q, err := rsmq.New()
	if err != nil {
		log.Fatalln(err)
	}
	ctx := context.Background()
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

	message, err := q.ReceiveMessage(ctx, rsmq.ReceiveQueueRequestOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Got result", message)
}

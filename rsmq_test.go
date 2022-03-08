package rsmq

import (
	"context"
	"log"
	"testing"
)

func TestSimple(t *testing.T) {
	q, err := New()
	if err != nil {
		log.Fatalln(err)
	}
	ctx := context.Background()
	qname := "SimpleGOTEST"
	err = q.CreateQueue(ctx, CreateQueueRequestOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	attributes, err := q.GetQueueAttributes(ctx, GetQueueAttributesOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Attributes found!:\n", attributes)

	uid, err := q.SendMessage(ctx, SendMessageRequestOptions{
		QName:   qname,
		Delay:   0,
		Message: "HELLO WORLD!",
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Send message with uid: ", uid)

	message, err := q.ReceiveMessage(ctx, ReceiveQueueRequestOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Got result", message)
}

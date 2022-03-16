package rsmq

import (
	"context"
	"log"
	"testing"
	"time"
)

func newQ(name string) (string, *RedisSMQ, context.Context, error) {
	qname := name + makeuid(4)
	q, err := New()
	if err != nil {
		log.Fatalln(err)
	}
	ctx := context.Background()
	err = q.CreateQueue(ctx, CreateQueueRequestOptions{QName: qname})
	if err != nil {
		log.Fatalln(err)
	}
	return qname, q, ctx, err
}
func TestList(t *testing.T) {
	_, q, ctx, err := newQ("listTest")
	if err != nil {
		t.Fatal(err)
	}
	r, err := q.ListQueues(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range r {
		if s == "OneMoreQueue" {
			found = true
		}
	}
	if !found {
		t.Log("Expected it to find OneMoreQueue")
		t.Fail()
	}
}

func TestDelete(t *testing.T) {
	qName, q, ctx, err := newQ("delete-test")
	if err != nil {
		t.Fatal(err)
	}

	r, err := q.ListQueues(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range r {
		if s == qName {
			found = true
		}
	}
	if !found {
		t.Log("Expected it to find ", qName)
		t.Fail()
	}
	err = q.DeleteQueue(ctx, DeleteQueueRequestOptions{QName: qName})
	if err != nil {
		t.Fatal(err)
	}
	r, err = q.ListQueues(ctx)
	if err != nil {
		t.Fatal(err)
	}
	secondFind := false
	for _, s := range r {
		if s == qName {
			secondFind = true
		}
	}
	if secondFind {
		t.Log("Expected not to find it after listing again")
		t.Fail()
	}
}

func TestSimple(t *testing.T) {
	qname, q, ctx, err := newQ("TestSimple")
	if err != nil {
		t.Fatal(err)
	}
	attributes, err := q.GetQueueAttributes(ctx, GetQueueAttributesOptions{QName: qname})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Attributes found!:\n", attributes)

	uid, err := q.SendMessage(ctx, SendMessageRequestOptions{
		QName:   qname,
		Delay:   0,
		Message: "HELLO WORLD!",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Send message with uid: ", uid)

	message, err := q.ReceiveMessage(ctx, ReceiveQueueRequestOptions{QName: qname})
	if err != nil {
		t.Fatal(err)
	}
	if message == nil {
		t.Log("Expected the test to receive a message but none was visibe")
		t.Fail()
	}
}

func TestDeleteMessage(t *testing.T) {
	qname, q, ctx, err := newQ("TestDeleteMessageQ")
	if err != nil {
		t.Fatal(err)
	}
	err = q.CreateQueue(ctx, CreateQueueRequestOptions{QName: qname})
	if err != nil {
		t.Fatal(err)
	}

	firstUID, err := q.SendMessage(ctx, SendMessageRequestOptions{
		QName:   qname,
		Delay:   0,
		Message: "HELLO WORLD!",
	})
	if err != nil {
		t.Fatal(err)
	}

	secondUID, err := q.SendMessage(ctx, SendMessageRequestOptions{
		QName:   qname,
		Delay:   0,
		Message: "HELLO WORLD!",
	})
	t.Log("Created second message", secondUID)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	err = q.DeleteMessage(ctx, DeleteMessageRequest{
		QName: qname,
		ID:    firstUID,
	})
	if err != nil {
		t.Fatal(err)
	}

	message, err := q.ReceiveMessage(ctx, ReceiveQueueRequestOptions{QName: qname})
	if err != nil {
		t.Fatalf("failed to recieve %v", err)
	}
	if message.ID == firstUID {
		t.Log("Expected not to receive the first UID because it should be deleted")
		t.Fail()
	}
}

func TestChangeMessageVisibilityMessage(t *testing.T) {
	qname, q, ctx, err := newQ("TestChangevisibility")

	firstUID, err := q.SendMessage(ctx, SendMessageRequestOptions{
		QName:   qname,
		Delay:   0,
		Message: "HELLO WORLD!",
	})
	if err != nil {
		log.Fatalln(err)
	}

	secondUID, err := q.SendMessage(ctx, SendMessageRequestOptions{
		QName:   qname,
		Delay:   0,
		Message: "HELLO WORLD!",
	})
	t.Log("Created second message", secondUID)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	err = q.ChangeMessageVisibility(ctx, ChangeMessageVisibilityOptions{
		QName:                    qname,
		ID:                       firstUID,
		VisibilityTimeoutSeconds: 1,
	})
	if err != nil {
		log.Fatalln(err)
	}

	message, err := q.ReceiveMessage(ctx, ReceiveQueueRequestOptions{QName: qname})
	if err != nil {
		t.Fatalf("failed to recieve %v", err)
	}
	if message.ID == firstUID {
		t.Log("Expected not to receive the first UID because it should be deleted")
		t.Fail()
	}

	// TODO: fix this slow test that depends on sleeping..
	time.Sleep(2 * time.Second)
	// not it should receive the hidden message from before
	message, err = q.ReceiveMessage(ctx, ReceiveQueueRequestOptions{QName: qname})
	if err != nil {
		t.Fatalf("failed to recieve %v", err)
	}
	if message.ID != firstUID {
		t.Log("Expected not to receive the first UID because it should be deleted")
		t.Fail()
	}
}

func TestPopMessage(t *testing.T) {
	qName, q, ctx, err := newQ("TestPopMessage")
	if err != nil {
		t.Fatal(err)
	}
	message, err := q.SendMessage(ctx, SendMessageRequestOptions{QName: qName, Message: "TEST MESSAGE BODY"})
	if err != nil {
		t.Fatal(err)
	}
	popMessage, err := q.PopMessage(ctx, PopMessageOptions{QName: qName})
	if err != nil {
		return
	}
	if message != popMessage.ID {
		t.Log("popped message should be the sent message")
		t.Fail()
	}

	popMessage2, err := q.PopMessage(ctx, PopMessageOptions{QName: qName})
	if err != nil {
		t.Fatal("should not error when calling PopMessage with no messages available", err)
	}
	if popMessage2 != nil {
		t.Log("Failed! it should be nil because no message")
		t.Fail()
	}

}

package rsmq

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
	"testing"
	"time"
)

func newQ(name string) (string, *RedisSMQ, context.Context, error) {
	qname := name + makeUID(4)

	ctx := context.Background()
	url := os.Getenv("REDIS_URL")
	if len(url) == 0 {
		url = "redis://localhost:6379"
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return "", nil, ctx, fmt.Errorf("parseURL: %w", err)
	}
	q, err := New(ctx, Options{
		Client: redis.NewClient(opts),
	})
	if err != nil {
		return "", nil, ctx, fmt.Errorf("new rsmq client: %w", err)
	}
	err = q.CreateQueue(ctx, CreateQueueRequestOptions{QName: qname})
	if err != nil {
		return "", nil, ctx, fmt.Errorf("create Queue %s: %w", qname, err)
	}
	return qname, q, ctx, nil
}
func TestList(t *testing.T) {
	qname, q, ctx, err := newQ("listTest")
	if err != nil {
		t.Fatal(err)
	}
	r, err := q.ListQueues(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range r {
		if s == qname {
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

	message, err := q.ReceiveMessage(ctx, ReceiveMessageOptions{QName: qname})
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

	message, err := q.ReceiveMessage(ctx, ReceiveMessageOptions{QName: qname})
	if err != nil {
		t.Fatalf("failed to recieve %v", err)
	}
	if message.ID == firstUID {
		t.Log("Expected not to receive the message because it should be deleted:\n", message)
		t.Fail()
	}
}

// fails about 30% of the time.. why??
func TestChangeMessageVisibilityMessage(t *testing.T) {
	qname, q, ctx, err := newQ("TestChangevisibility")

	msg, err := q.ReceiveMessage(ctx, ReceiveMessageOptions{
		QName:             qname,
		VisibilityTimeout: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg != nil {
		t.Log("should not have returned a message on a new Q")
		t.Fail()
	}

	ok, err := q.ChangeMessageVisibility(ctx, ChangeMessageVisibilityOptions{
		QName:             qname,
		ID:                "fakeUIDofMessage",
		VisibilityTimeout: 2,
	})
	if ok {
		t.Log("Message visibility should return false if the message doesn't currently exist ")
		t.FailNow()
	}
	firstUID, err := q.SendMessage(ctx, SendMessageRequestOptions{
		QName:   qname,
		Delay:   0,
		Message: "HELLO WORLD!",
	})
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	ok, err = q.ChangeMessageVisibility(ctx, ChangeMessageVisibilityOptions{
		QName:             qname,
		ID:                firstUID,
		VisibilityTimeout: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Log("Message visibility request should be successful ")
		t.FailNow()
	}
	message, err := q.ReceiveMessage(ctx, ReceiveMessageOptions{QName: qname})
	if err != nil {
		t.Fatalf("failed to recieve %v", err)
	}
	if message != nil {
		t.Logf("It should not have returned any message first message on [%s] but we got:\n %s", qname, message)
		t.FailNow()
	}

	// setting visibility to 0 should make the message receivable immediately
	ok, err = q.ChangeMessageVisibility(ctx, ChangeMessageVisibilityOptions{
		QName:             qname,
		ID:                firstUID,
		VisibilityTimeout: 0,
	})
	if err != nil {
		t.Fatal("Did not expect to fail changing visibility", err)
	}
	if !ok {
		t.Log("Message visibility request should be succesful ")
		t.FailNow()
	}
	time.Sleep(50 * time.Millisecond)
	// It should be able to request this
	message, err = q.ReceiveMessage(ctx, ReceiveMessageOptions{QName: qname})
	if err != nil {
		t.Fatalf("failed to recieve %v", err)
	}
	if message == nil || message.ID != firstUID {
		t.Logf("It should have returned the first message on [%s] but we got:\n %s", qname, message)
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

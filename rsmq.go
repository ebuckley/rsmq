package rsmq

import "time"

type Message struct {
	ID string
	Message string
	RC int
	FR time.Time
	Sent time.Time
}

type RedisSMQ struct {

}

type RecieveQueueRequestOptions struct {
	QName string
}

type CreateQueueRequestOptions struct {
	QName string
}
type GetQueueAttributesOptions struct {
	QName string
}

type QueueAttributes struct {
	VisibilityTimeout time.Duration
	DelayForMessages time.Duration
	MaxSizeBytes int
	TotalReceived int
	Created time.Time
	Modified time.Time
	CurrentN int
	HiddenMessages int
}

func (rsmq RedisSMQ) CreateQueue(opts CreateQueueRequestOptions) error  {
	return nil
}

func (rsmq RedisSMQ) RecieveMessage(opts RecieveQueueRequestOptions) (error, Message) {
	return nil, Message{}
}

func (rsmq RedisSMQ) GetQueueAttributes(opts GetQueueAttributesOptions) (error, QueueAttributes) {
	return nil, QueueAttributes{}
}

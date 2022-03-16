package rsmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"math/rand"
	"strconv"
	"time"
)

type Message struct {
	// ID is the internal message identifier
	ID string
	// Message is the contents of the message
	Message string
	// RC is the number of times this message was received
	RC int64
	// FR is the time when this message was first received
	FR time.Time
	// Sent is the time when this message ws first sent
	Sent time.Time
}

type ReceiveQueueRequestOptions struct {
	QName string
	// TODO implement the visibility timeout override
}

type CreateQueueRequestOptions struct {
	QName string
}
type GetQueueAttributesOptions struct {
	QName string
}
type DeleteQueueRequestOptions struct {
	QName string
}

type SendMessageRequestOptions struct {
	QName   string
	Delay   int
	Message string
}

type ChangeMessageVisibilityOptions struct {
	QName                    string
	ID                       string
	VisibilityTimeoutSeconds int
}

type DeleteMessageRequest struct {
	QName string
	ID    string
}

type QueueAttributes struct {
	VisibilityTimeout int    `redis:"vt"`
	DelayForMessages  int    `redis:"Delay"`
	MaxSizeBytes      int64  `redis:"maxsize"`
	TotalReceived     int64  `redis:"totalrecv"`
	TotalSent         int64  `redis:"totalsent"`
	Created           string `redis:"created"`
	Modified          string `redis:"modified"`
	CurrentN          int64
	HiddenMessages    int64
}

func (q QueueAttributes) String() string {
	marshal, err := json.Marshal(q)
	if err != nil {
		return fmt.Sprintf("Could not marshal QueueAttributes: %s", err)
	}
	return string(marshal)
}

type PopMessageOptions struct {
	QName string
}

type qAttr struct {
	VisibilityTimeout int   `redis:"vt"`
	DelayForMessages  int   `redis:"delay"`
	MaxSizeBytes      int64 `redis:"maxsize"`
	TimeSent          time.Time
	UID               string
}

func (q qAttr) timeSentUnix() string {
	return strconv.FormatInt(q.TimeSent.UnixMilli(), 10)
}
func (q qAttr) timeVisibilityExpiresUnix() string {
	return strconv.FormatInt(q.TimeSent.UnixMilli()+int64(q.VisibilityTimeout*1000), 10)
}

type RedisSMQ struct {
	cl                 *redis.Client
	popMessageSha1     *string
	receiveMessageSha1 *string
	hideMessageSha1    *string
	ns                 string
}

func (rsmq *RedisSMQ) CreateQueue(ctx context.Context, opts CreateQueueRequestOptions) error {
	// default vt to be 30
	// default delay to be 0
	// default maxsize to be 65536
	key := rsmq.ns + ":" + opts.QName + ":Q"

	result, err := rsmq.cl.Time(ctx).Result()
	if err != nil {
		return fmt.Errorf("CreateQueue: %w", err)
	}
	_, err = rsmq.cl.HMSet(ctx, key, map[string]interface{}{
		"createdby": "ersin",
		"vt":        30,
		"delay":     0,
		"maxsize":   65536,
		"created":   result,
		"modified":  result,
	}).Result()
	if err != nil {
		return fmt.Errorf("CreateQueue: set queue params: %w", err)
	}
	_, err = rsmq.cl.SAdd(ctx, rsmq.ns+":QUEUES", opts.QName).Result()
	if err != nil {
		return fmt.Errorf("CreateQueue: add queue to QUEUES set: %w", err)
	}

	return nil
}

// ReceiveMessage receives the next message from the queue, re-entering the queue if it is not received elsewhere
// A received message is invisible to other consumers for an amount of time
func (rsmq *RedisSMQ) ReceiveMessage(ctx context.Context, opts ReceiveQueueRequestOptions) (*Message, error) {
	key := rsmq.ns + ":" + opts.QName
	q, err := rsmq.getQueue(ctx, opts.QName)
	if err != nil {
		return nil, fmt.Errorf("recieve message: %w", err)
	}
	timeSentUnix := strconv.FormatInt(q.TimeSent.UnixMilli(), 10)
	timeVisibilityExpiresUnix := strconv.FormatInt(q.TimeSent.UnixMilli()+int64(q.VisibilityTimeout*1000), 10)
	// TODO -- potential panic if messageSHA1 is nil
	results, err := rsmq.cl.EvalSha(ctx, *rsmq.receiveMessageSha1, []string{key, timeSentUnix, timeVisibilityExpiresUnix}).Slice()
	if err != nil {
		return nil, fmt.Errorf("recieve message: eval recieveMessage script: %w", err)
	}

	return unmarshalMessage(results, q)
}

func (rsmq *RedisSMQ) getQueue(ctx context.Context, name string) (*qAttr, error) {
	key := rsmq.ns + ":" + name + ":Q"
	pipe := rsmq.cl.Pipeline()
	t := pipe.Time(ctx)

	attr := pipe.HMGet(ctx, key, "vt", "delay", "maxsize")
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("getQ %s: %w", key, err)
	}
	var q qAttr
	err = attr.Scan(&q)
	if err != nil {
		return nil, err
	}

	q.TimeSent = t.Val()
	q.UID = makeuid(22)

	return &q, nil
}

func (rsmq *RedisSMQ) SendMessage(ctx context.Context, opts SendMessageRequestOptions) (string, error) {
	key := rsmq.ns + ":" + opts.QName
	q, err := rsmq.getQueue(ctx, opts.QName)
	if err != nil {
		return "", err
	}
	if int64(len(opts.Message)) > q.MaxSizeBytes {
		return "", errors.New("Message is larger than allowed max size: " + strconv.FormatInt(q.MaxSizeBytes, 10))
	}
	pipe := rsmq.cl.Pipeline()
	sendTime := time.Duration(q.DelayForMessages) * time.Millisecond
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(q.TimeSent.Add(sendTime).UnixMilli()),
		Member: q.UID,
	})
	pipe.HSet(ctx, key+":Q", q.UID, opts.Message)
	pipe.HIncrBy(ctx, key+":Q", "totalsent", 1)
	// TODO if realtime Q then run 'zcard key'
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("sending message to Q: %w", err)
	}

	return q.UID, nil
}

func (rsmq *RedisSMQ) GetQueueAttributes(ctx context.Context, opts GetQueueAttributesOptions) (*QueueAttributes, error) {
	key := rsmq.ns + ":" + opts.QName
	t, err := rsmq.cl.Time(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("GetQueueAttributes: %w", err)
	}

	pipe := rsmq.cl.Pipeline()
	fields := []string{"vt", "delay", "maxsize", "totalrecv", "totalsent", "created", "modified"}
	queueAttrs := pipe.HMGet(ctx, rsmq.ns+":"+opts.QName+":Q", fields...)

	count := pipe.ZCard(ctx, key)
	// TODO validate this is right level or do we need UnixMilli/UnixNano
	zcount := pipe.ZCount(ctx, key, fmt.Sprint(t.UnixMilli()), "+inf")

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetQueueAttributes: %w", err)
	}
	var attr QueueAttributes
	err = queueAttrs.Scan(&attr)
	if err != nil {
		return nil, fmt.Errorf("GetQueueAttributes: %w", err)
	}

	attr.CurrentN = count.Val()
	attr.HiddenMessages = zcount.Val()

	return &attr, nil
}

func (rsmq *RedisSMQ) DeleteQueue(ctx context.Context, options DeleteQueueRequestOptions) error {
	if len(options.QName) == 0 {
		return errors.New("QName is empty")
	}
	key := rsmq.ns + ":" + options.QName
	pipe := rsmq.cl.Pipeline()

	pipe.Del(ctx, key)
	pipe.SRem(ctx, rsmq.ns+":QUEUES", options.QName)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("DeleteQueue: %w", err)
	}
	return nil
}

func (rsmq *RedisSMQ) DeleteMessage(ctx context.Context, options DeleteMessageRequest) error {
	if len(options.QName) == 0 || len(options.ID) == 0 {
		return errors.New("options.QNAME or options.ID was empty but it should not be empty")
	}
	key := rsmq.ns + ":" + options.QName
	pipe := rsmq.cl.Pipeline()
	pipe.ZRem(ctx, key, options.ID)
	pipe.HDel(ctx, key+":Q", options.ID+":rc", options.ID+":fr")
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("deleteMessage: %w", err)
	}
	return nil
}

func (rsmq *RedisSMQ) ChangeMessageVisibility(ctx context.Context, options ChangeMessageVisibilityOptions) error {
	if len(options.QName) == 0 || len(options.ID) == 0 {
		return fmt.Errorf("ChangeMessageVisibility requires QName and ID parameters")
	}
	q, err := rsmq.getQueue(ctx, options.QName)
	if err != nil {
		return fmt.Errorf("getQueue: %w", err)
	}
	args := []string{
		rsmq.ns + ":" + options.QName,
		options.ID,
		q.timeSentUnix(),
		q.timeVisibilityExpiresUnix(),
	}
	_, err = rsmq.cl.EvalSha(ctx, *rsmq.hideMessageSha1, args).Result()
	if err != nil {
		return fmt.Errorf("eval hideMessageSha1: %w", err)
	}
	return nil
}

func (rsmq *RedisSMQ) ListQueues(ctx context.Context) ([]string, error) {
	result, err := rsmq.cl.SMembers(ctx, rsmq.ns+":"+"QUEUES").Result()
	return result, err
}

// PopMessage will Receive the next message from the queue and delete it.
//
// Important: This method deletes the message it receives right away.
// There is no way to receive the message again if something goes wrong while working on the message.
func (rsmq *RedisSMQ) PopMessage(ctx context.Context, options PopMessageOptions) (*Message, error) {
	if len(options.QName) == 0 {
		return nil, errors.New("popMessage validation failed. Expected options.QName to be set")
	}
	q, err := rsmq.getQueue(ctx, options.QName)
	if err != nil {
		return nil, err
	}

	res, err := rsmq.cl.EvalSha(ctx, *rsmq.popMessageSha1, []string{rsmq.ns + ":" + options.QName, q.timeSentUnix()}).Slice()
	if err != nil {
		return nil, fmt.Errorf("popMessage evalSha: %w", err)
	}
	return unmarshalMessage(res, q)
}

func (rsmq *RedisSMQ) SetQueueAttributes() error {
	panic(any("not implemented"))
}

func (rsmq *RedisSMQ) Close() error {
	return rsmq.cl.Close()
}

func (rsmq *RedisSMQ) initScripts(ctx context.Context) error {
	popMessage := rsmq.cl.ScriptLoad(ctx, scriptPopMessage)
	popMessageSha1, err := popMessage.Result()
	if err != nil {
		return fmt.Errorf("init scriptPopMessage: %w", err)
	}
	rsmq.popMessageSha1 = &popMessageSha1

	receiveMessage := rsmq.cl.ScriptLoad(ctx, scriptReceiveMessage)
	receiveMessageSha1, err := receiveMessage.Result()
	if err != nil {
		return fmt.Errorf("init script_recieveMessage: %w", err)
	}
	rsmq.receiveMessageSha1 = &receiveMessageSha1

	changeVisMessage := rsmq.cl.ScriptLoad(ctx, scriptChangeMessageVisibility)
	hideMessageSha1, err := changeVisMessage.Result()
	if err != nil {
		return fmt.Errorf("init scriptChangeMessageVisibility: %w", err)
	}
	rsmq.hideMessageSha1 = &hideMessageSha1
	return nil
}

// New creates the RedisSMQ with default params
func New() (*RedisSMQ, error) {
	cl := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	rq := &RedisSMQ{cl: cl, ns: "rsmq"}
	err := rq.initScripts(context.Background())
	if err != nil {
		return nil, err
	}
	return rq, nil
}

func unmarshalMessage(results []interface{}, q *qAttr) (*Message, error) {
	// an empty result set means no messages are available
	if len(results) == 0 {
		return nil, nil
	}
	if len(results) != 4 {
		return nil, fmt.Errorf("unexpected result set, expected 4 items but got %v", results)
	}
	uid, ok := results[0].(string)
	if !ok {
		return nil, fmt.Errorf("could not serialize string type from first element")
	}
	msg, ok := results[1].(string)
	if !ok {
		return nil, fmt.Errorf("could not serialize string type from second element")
	}
	rc, ok := results[2].(int64)
	if !ok {
		return nil, fmt.Errorf("could not serialize int64 type from third element")
	}
	ts, ok := results[3].(string)
	if !ok {
		return nil, fmt.Errorf("could not serialize timestamp string type from fourth element")
	}
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse the timestamp string: %w", err)
	}

	return &Message{
		ID:      uid,
		Message: msg,
		RC:      rc,
		FR:      time.UnixMilli(tsInt),
		Sent:    q.TimeSent,
	}, nil
}

// makeuid returns an ID for a string
// TODO use cryptographic random!!
func makeuid(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

const scriptPopMessage = `local msg = redis.call("ZRANGEBYSCORE", KEYS[1], "-inf", KEYS[2], "LIMIT", "0", "1")
			if #msg == 0 then
				return {}
			end
			redis.call("HINCRBY", KEYS[1] .. ":Q", "totalrecv", 1)
			local mbody = redis.call("HGET", KEYS[1] .. ":Q", msg[1])
			local rc = redis.call("HINCRBY", KEYS[1] .. ":Q", msg[1] .. ":rc", 1)
			local o = {msg[1], mbody, rc}
			if rc==1 then
				table.insert(o, KEYS[2])
			else
				local fr = redis.call("HGET", KEYS[1] .. ":Q", msg[1] .. ":fr")
				table.insert(o, fr)
			end
			redis.call("ZREM", KEYS[1], msg[1])
			redis.call("HDEL", KEYS[1] .. ":Q", msg[1], msg[1] .. ":rc", msg[1] .. ":fr")
			return o`
const scriptReceiveMessage = `local msg = redis.call("ZRANGEBYSCORE", KEYS[1], "-inf", KEYS[2], "LIMIT", "0", "1")
			if #msg == 0 then
				return {}
			end
			redis.call("ZADD", KEYS[1], KEYS[3], msg[1])
			redis.call("HINCRBY", KEYS[1] .. ":Q", "totalrecv", 1)
			local mbody = redis.call("HGET", KEYS[1] .. ":Q", msg[1])
			local rc = redis.call("HINCRBY", KEYS[1] .. ":Q", msg[1] .. ":rc", 1)
			local o = {msg[1], mbody, rc}
			if rc==1 then
				redis.call("HSET", KEYS[1] .. ":Q", msg[1] .. ":fr", KEYS[2])
				table.insert(o, KEYS[2])
			else
				local fr = redis.call("HGET", KEYS[1] .. ":Q", msg[1] .. ":fr")
				table.insert(o, fr)
			end
			return o`
const scriptChangeMessageVisibility = `local msg = redis.call("ZSCORE", KEYS[1], KEYS[2])
			if not msg then
				return 0
			end
			redis.call("ZADD", KEYS[1], KEYS[3], KEYS[2])
			return 1`

package worker

import (
	"context"
	"errors"
	"github.com/ebuckley/rsmq/q"
	"time"
)

// Worker is an experimental framework built on top of RSMQ that is modelled after the nodejs rsmq-worker project
type Worker struct {
	cl      *q.RedisSMQ
	work    chan *q.Message
	quit    chan bool
	ctx     context.Context
	qName   string
	handler Handler
}

type Handler interface {
	Message(ctx context.Context, msg *q.Message) (bool, error)
	Error(ctx context.Context, err error, msg *q.Message) error
	DeadlinePassed(ctx context.Context, msg *q.Message) error
}

func New(ctx context.Context, cl *q.RedisSMQ, qName string, handler Handler) *Worker {
	work := make(chan *q.Message, 0)
	quit := make(chan bool, 0)
	_ = cl.CreateQueue(ctx, q.CreateQueueRequestOptions{QName: qName})
	return &Worker{cl: cl, work: work, quit: quit, ctx: ctx, handler: handler, qName: qName}
}

func (w *Worker) Start() {
	workers := 10

	// receiver goroutine started
	go func() {
		for {
			message, err := w.cl.ReceiveMessage(w.ctx, q.ReceiveMessageOptions{
				QName: w.qName,
			})
			if err != nil {
				return
			}
			if message != nil {
				w.work <- message
			}
			time.Sleep(time.Second)
		}
	}()

	// workers
	for ; workers > 0; workers-- {
		go func(ctx context.Context, id int) {
			for {
				select {
				case <-w.quit:
					return
				case msg := <-w.work:
					iCtx, cancel := context.WithDeadline(ctx, *msg.Deadline)
					ok, err := w.handler.Message(iCtx, msg)
					if errors.Is(err, context.DeadlineExceeded) {
						err := w.handler.DeadlinePassed(ctx, msg)
						if err != nil {
							// TODO internal error?
						}
					} else if err != nil {

						_, err := w.cl.ChangeMessageVisibility(ctx, q.ChangeMessageVisibilityOptions{
							QName:             w.qName,
							ID:                msg.ID,
							VisibilityTimeout: 0,
						})
						if err != nil {
							// TODO log this or something?
						}
						err = w.handler.Error(ctx, err, msg)
						if err != nil {
							// TODO what if the error handler has an error!
						}
					}
					if ok {
						err = w.cl.DeleteMessage(ctx, q.DeleteMessageRequest{
							QName: w.qName,
							ID:    msg.ID,
						})
						if err != nil {
							// TODO log this or something?
						}
					}
					cancel()
				}
			}
		}(w.ctx, workers)
	}
	// block until a quit signal is received
	<-w.quit
}

func (w *Worker) Quit() error {
	err := w.cl.Close()
	if err != nil {
		return err
	}
	w.quit <- true
	return nil
}

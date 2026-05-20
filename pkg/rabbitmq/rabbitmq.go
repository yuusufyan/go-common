package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type RabbitMQClient interface {
	Publish(ctx context.Context, queueName string, body []byte) error
	Consume(queueName string, handler func(ctx context.Context, body []byte) error) error
	Close() error
}

type rabbitMQClient struct {
	url    string
	conn   *amqp.Connection
	pubCh  *amqp.Channel
	consCh *amqp.Channel
	mu     sync.RWMutex
	closed bool
	log    *logrus.Entry
}

func NewRabbitMQClient(url string) (RabbitMQClient, error) {
	client := &rabbitMQClient{
		url: url,
		log: logrus.WithField("module", "rabbitmq"),
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	return client, nil
}

func (r *rabbitMQClient) connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connectLocked()
}

// connectLocked performs the actual connection. Caller must hold r.mu.Lock().
func (r *rabbitMQClient) connectLocked() error {
	conn, err := amqp.Dial(r.url)
	if err != nil {
		return err
	}

	pubCh, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	consCh, err := conn.Channel()
	if err != nil {
		pubCh.Close()
		conn.Close()
		return err
	}

	r.conn = conn
	r.pubCh = pubCh
	r.consCh = consCh

	// Listen for connection closure to trigger auto-reconnect
	go func() {
		closeChan := r.conn.NotifyClose(make(chan *amqp.Error))
		amqpErr := <-closeChan

		r.mu.RLock()
		isClosed := r.closed
		r.mu.RUnlock()

		if amqpErr != nil && !isClosed {
			r.log.Errorf("Connection closed: %v. Reconnecting...", amqpErr)
			for {
				time.Sleep(5 * time.Second)
				// Acquire lock before calling connectLocked to avoid deadlock
				r.mu.Lock()
				err := r.connectLocked()
				r.mu.Unlock()
				if err == nil {
					r.log.Info("Reconnected successfully")
					break
				}
				r.log.Warn("Reconnect failed, retrying...")
			}
		}
	}()

	return nil
}

func (r *rabbitMQClient) Publish(ctx context.Context, queueName string, body []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pubCh == nil || r.pubCh.IsClosed() {
		return fmt.Errorf("rabbitMQ channel is closed")
	}

	_, err := r.pubCh.QueueDeclare(
		queueName,
		true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	return r.pubCh.PublishWithContext(ctx,
		"", queueName, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		})
}

func (r *rabbitMQClient) Consume(queueName string, handler func(ctx context.Context, body []byte) error) error {
	// Re-declaration loop to handle reconnection
	go func() {
		for {
			r.mu.RLock()
			isClosed := r.closed
			currConsCh := r.consCh
			r.mu.RUnlock()

			if isClosed {
				break
			}

			if currConsCh == nil || currConsCh.IsClosed() {
				time.Sleep(2 * time.Second)
				continue
			}

			// Setup DLQ and Main Queue
			dlxName := queueName + ".dlx"
			dlqName := queueName + ".dlq"
			
			if err := currConsCh.ExchangeDeclare(dlxName, "direct", true, false, false, false, nil); err != nil {
				r.log.Errorf("Failed to declare DLX '%s': %v", dlxName, err)
				time.Sleep(5 * time.Second)
				continue
			}
			
			if _, err := currConsCh.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
				r.log.Errorf("Failed to declare DLQ '%s': %v", dlqName, err)
				time.Sleep(5 * time.Second)
				continue
			}
			
			if err := currConsCh.QueueBind(dlqName, dlqName, dlxName, false, nil); err != nil {
				r.log.Errorf("Failed to bind DLQ '%s': %v", dlqName, err)
				time.Sleep(5 * time.Second)
				continue
			}

			args := amqp.Table{"x-dead-letter-exchange": dlxName, "x-dead-letter-routing-key": dlqName}
			q, err := currConsCh.QueueDeclare(queueName, true, false, false, false, args)
			if err != nil {
				r.log.Errorf("Failed to declare queue '%s': %v", queueName, err)
				time.Sleep(5 * time.Second)
				continue
			}

			currConsCh.Qos(1, 0, false)
			msgs, err := currConsCh.Consume(q.Name, "", false, false, false, false, nil)
			if err != nil {
				r.log.Errorf("Failed to start consuming '%s': %v", queueName, err)
				time.Sleep(5 * time.Second)
				continue
			}

			r.log.Infof("Consumer started for '%s'", queueName)

			for d := range msgs {
				func() {
					// Add Timeout to prevent hanging workers
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					defer cancel()

					defer func() {
						if rec := recover(); rec != nil {
							r.log.Errorf("PANIC in '%s': %v", queueName, rec)
							d.Nack(false, false)
						}
					}()

					if err := handler(ctx, d.Body); err != nil {
						r.log.Errorf("Handler error in '%s': %v. Sent to DLQ.", queueName, err)
						d.Nack(false, false)
					} else {
						d.Ack(false)
					}
				}()
			}

			r.log.Warnf("Consumer for '%s' closed. Attempting to restart...", queueName)
			time.Sleep(2 * time.Second)
		}
	}()

	return nil
}

func (r *rabbitMQClient) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.closed = true
	if r.consCh != nil {
		r.consCh.Close()
	}
	if r.pubCh != nil {
		r.pubCh.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

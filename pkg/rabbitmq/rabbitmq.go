package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rmscoal/dalc/config"
)

type RabbitMQ struct {
	Conn *amqp.Connection
	Ch   *amqp.Channel

	TaskQueue amqp.Queue
}

func New(cfg config.RabbitMQ) *RabbitMQ {
	rburl, err := url.Parse(fmt.Sprintf("amqp://%s/%s", cfg.Host, cfg.VirtualHost))
	if err != nil {
		log.Fatal("unable to parse base rabbitmq dsn", "err", err)
	}
	rburl.User = url.UserPassword(cfg.Username, cfg.Password)

	rabbit := &RabbitMQ{}
	conn, err := amqp.Dial(rburl.String())
	if err != nil {
		log.Fatal("unable to connect to rabbitmq:", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("unable to receive channel from rabbitmq:", err)
	}

	rabbit.Conn = conn
	rabbit.Ch = ch

	rabbit.setup()

	return rabbit
}

func (r *RabbitMQ) setup() {
	var err error

	// Create exchange
	err = r.Ch.ExchangeDeclare(
		"tasks",  // name
		"direct", // kind
		true,     // durable
		false,    // auto-delete
		false,    // internal
		false,    // nowait
		nil,
	)
	if err != nil {
		log.Fatal("unable to declare exchange:", err)
	}

	// Create queue
	r.TaskQueue, err = r.Ch.QueueDeclare(
		"tasks",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("unable to declare queue:", err)
	}

	// Binds the tasks queue to the tasks exchange
	err = r.Ch.QueueBind(
		"tasks", // queue name
		"tasks", // routing key
		"tasks", // exchange name
		false,   // no wait
		nil,
	)
	if err != nil {
		log.Fatal("unable to bind the queue and exchange:", err)
	}
}

func (r *RabbitMQ) Shutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("unable to shutdown rabbitmq")
	default:
		err := errors.Join(r.Ch.Close(), r.Conn.Close())
		return err
	}
}

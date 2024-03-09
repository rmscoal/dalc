package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/charmbracelet/log"
	"github.com/google/uuid"

	"github.com/rmscoal/dalc/config"
	"github.com/rmscoal/dalc/pkg/postgres"
	rabbitmq "github.com/rmscoal/dalc/pkg/rabbitmq"
	"github.com/rmscoal/dalc/shared/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	rabbit   *rabbitmq.RabbitMQ
	pg       *postgres.Postgres
	workerID string = uuid.NewString()
)

func main() {
	cfg := config.GetConfig("config.yaml")
	appCtx := context.Background()

	rabbit = rabbitmq.New(cfg.RabbitMQ.URL) // We have our rabbitmq ready to use
	pg = postgres.New(cfg.Database.URL)

	tasks, err := rabbit.Ch.ConsumeWithContext(
		appCtx,
		"tasks",
		workerID,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Fatal("unable to consume tasks:", err)
	}

	// Listens for new incoming tasks
	log.Info("worker started")
	go subscribe(appCtx, pg, tasks)

	// Listens for and handles quit
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(appCtx, 10*time.Second)
	defer cancel()
	log.Infof("Shutting down worker...")
	rabbit.Shutdown(ctx)
	pg.Shutdown(ctx)
}

func subscribe(ctx context.Context, pg *postgres.Postgres, tasks <-chan amqp.Delivery) {
	for task := range tasks {
		log.Infof("task with messageId %s received", task.MessageId)

		var err error
		var tm domain.Task

		err = json.Unmarshal(task.Body, &tm)
		if err != nil {
			go log.Error("unable to decode task body:", err)
		}

		expression, err := govaluate.NewEvaluableExpression(tm.Expression)
		if err != nil {
			tm.Status = domain.FAILED
		}
		result, err := expression.Evaluate(nil)
		if err != nil {
			tm.Status = domain.FAILED
		}
		switch r := result.(type) {
		case float64:
			tm.Result = &r
		case float32:
			x := float64(r)
			tm.Result = &x
		case int64:
			x := float64(r)
			tm.Result = &x
		case int32:
			x := float64(r)
			tm.Result = &x
		default:
			tm.Status = domain.FAILED
		}

		_, err = pg.DB.ExecContext(ctx, `UPDATE tasks SET result = $1, status = $2 WHERE id = $3`, tm.Result, tm.Status, tm.ID)
		if err != nil {
			go log.Error("unable to update task result:", err)
		}

		err = task.Ack(false)
		if err != nil {
			go log.Error("unable to ack task:", err)
		}
	}
}

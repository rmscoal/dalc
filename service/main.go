package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rmscoal/dalc/config"
	"github.com/rmscoal/dalc/pkg/postgres"
	"github.com/rmscoal/dalc/pkg/rabbitmq"
	"github.com/rmscoal/dalc/shared/message"
)

func main() {
	cfg := config.GetConfig("config.yaml")

	rabbit := rabbitmq.New(cfg.RabbitMQ.URL)
	pg := postgres.New(cfg.Database.URL)

	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", newTaskHandler(pg, rabbit))

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start the server
	log.Infof("Service listening on %s\n", server.Addr)
	go func(server *http.Server) {
		if err := server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				log.Fatal("unable to start http server")
			}
		}
	}(server)

	// Listens for and handles quit
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Infof("Shutting down service...")
	rabbit.Shutdown(ctx)
	pg.Shutdown(ctx)
	server.Shutdown(ctx)
}

func newTaskHandler(pg *postgres.Postgres, rabbit *rabbitmq.RabbitMQ) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("method not allowed"))
		}

		var req message.TaskMessage

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(422)
			w.Write([]byte(err.Error()))
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		req.Status = message.SCHEDULED

		// Save to database
		err = pg.DB.QueryRowContext(
			r.Context(),
			`INSERT INTO tasks (expression, status) VALUES ($1, $2) RETURNING id`,
			req.Expression, req.Status,
		).Scan(&req.ID)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		// Marshal for our rabbitmq payload
		data, err := json.Marshal(req)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		// Publish to rabbitmq
		err = rabbit.Ch.PublishWithContext(
			r.Context(),
			"tasks",
			"tasks",
			false,
			false,
			amqp.Publishing{
				// Will be used for tracing purposes
				// Headers:         map[string]interface{}{},
				ContentType: "application/json",
				MessageId:   uuid.NewString(),
				Timestamp:   time.Now(),
				Body:        data,
			})
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(201)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("Your task has been sent to our workers"))
		return
	}
}

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
	"github.com/rmscoal/dalc/shared/domain"
)

func main() {
	cfg := config.GetConfig("config.yaml")

	rabbit := rabbitmq.New(cfg.RabbitMQ.URL)
	pg := postgres.New(cfg.Database.URL)

	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", taskHandler(pg, rabbit))

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

func taskHandler(pg *postgres.Postgres, rabbit *rabbitmq.RabbitMQ) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			id := r.URL.Query().Get("id")
			if id == "" {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("please provide id"))
				return
			}

			var task domain.Task
			err := pg.DB.QueryRowContext(r.Context(),
				`SELECT id, expression, status, result FROM tasks WHERE id = $1`,
				id).Scan(
				&task.ID,
				&task.Expression,
				&task.Status,
				&task.Result)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			body, _ := json.Marshal(task)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write(body)
			return
		} else if r.Method == http.MethodPost {
			var req domain.Task

			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(422)
				w.Write([]byte(err.Error()))
				return
			}

			err = json.Unmarshal(body, &req)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(400)
				w.Write([]byte(err.Error()))
				return
			}

			req.Status = domain.SCHEDULED

			// Save to database
			err = pg.DB.QueryRowContext(
				r.Context(),
				`INSERT INTO tasks (expression, status) VALUES ($1, $2) RETURNING id`,
				req.Expression, req.Status,
			).Scan(&req.ID)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}

			// Marshal for our rabbitmq payload
			data, err := json.Marshal(req)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
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
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(201)
			w.Write([]byte(fmt.Sprintf("Your task has been sent to our workers with %d", req.ID)))
			return
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("method not allowed"))
			return
		}
	}
}

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
	"github.com/rmscoal/dalc/shared/api"
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
	log.Info("Web service started,", "address", server.Addr)
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
	log.Info("Shutting down service...")
	rabbit.Shutdown(ctx)
	pg.Shutdown(ctx)
	server.Shutdown(ctx)
}

func taskHandler(pg *postgres.Postgres, rabbit *rabbitmq.RabbitMQ) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getTask(pg, w, r)
		} else if r.Method == http.MethodPost {
			createTask(pg, rabbit, w, r)
		} else {
			api.NewErrorMessage(w, fmt.Errorf("invalid method"), http.StatusMethodNotAllowed)
		}
	}
}

func getTask(pg *postgres.Postgres, w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		api.NewErrorMessage(w, fmt.Errorf("missing id"), http.StatusBadRequest)
		return
	}

	var task domain.Task
	err := pg.DB.QueryRowContext(r.Context(),
		`SELECT id, expression, status, result, error_message FROM tasks WHERE id = $1`,
		id).Scan(
		&task.ID,
		&task.Expression,
		&task.Status,
		&task.Result,
		&task.ErrorMessage)
	if err != nil {
		api.NewErrorMessage(w, err, http.StatusNotFound)
		return
	}

	body, _ := json.Marshal(task)
	api.NewOk(w, body)
}

func createTask(pg *postgres.Postgres, rabbit *rabbitmq.RabbitMQ, w http.ResponseWriter, r *http.Request) {
	var req domain.Task

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.NewErrorMessage(w, err, http.StatusUnprocessableEntity)
		return
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		api.NewErrorMessage(w, err, http.StatusBadRequest)
		return
	}

	req.Status = domain.SCHEDULED

	// Save to database
	err = pg.DB.QueryRowContext(r.Context(),
		`INSERT INTO tasks (expression, status) VALUES ($1, $2) RETURNING id`,
		req.Expression, req.Status,
	).Scan(&req.ID)
	if err != nil {
		api.NewErrorMessage(w, err, http.StatusInternalServerError)
		return
	}

	// Marshal for our rabbitmq payload
	data, err := json.Marshal(req)
	if err != nil {
		api.NewErrorMessage(w, err, http.StatusInternalServerError)
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
		api.NewErrorMessage(w, err, http.StatusInternalServerError)
		return
	}

	api.NewOkMessage(w, fmt.Sprintf("Your task has been sent to our workers with %d", req.ID))
}

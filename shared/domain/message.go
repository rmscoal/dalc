package domain

type TaskStatus string

const (
	SCHEDULED TaskStatus = "SCHEDULED"
	FAILED    TaskStatus = "FAILED"
	COMPLETED TaskStatus = "COMPLETED"
)

type Task struct {
	ID           int        `json:"id"`
	Expression   string     `json:"expression"`
	Status       TaskStatus `json:"status"`
	Result       *float64   `json:"result"`
	ErrorMessage string     `json:"errorMessage"`
}

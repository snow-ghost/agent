package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/snow-ghost/agent/core"
)

// Ingestor is a simple in-memory queue and optional HTTP endpoint to submit tasks.
type Ingestor struct {
	mu    sync.Mutex
	queue []core.Task
	solve func(context.Context, core.Task) (core.Result, error)
}

func NewIngestor(solve func(context.Context, core.Task) (core.Result, error)) *Ingestor {
	return &Ingestor{solve: solve, queue: make([]core.Task, 0)}
}

func (i *Ingestor) Enqueue(t core.Task) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.queue = append(i.queue, t)
}

func (i *Ingestor) Dequeue() (core.Task, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if len(i.queue) == 0 {
		return core.Task{}, false
	}
	t := i.queue[0]
	i.queue = i.queue[1:]
	return t, true
}

// ServeHTTP handles POST /solve with JSON Task and returns Result.
func (i *Ingestor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var t core.Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := i.solve(r.Context(), t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

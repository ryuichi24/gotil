 package todo

import (
	"errors"
	"sync"
)

// TodoRepository defines the interface for todo data access operations
type TodoRepository interface {
	FindAll() ([]TodoModel, error)
	FindByID(id string) (*TodoModel, error)
	Create(todo TodoModel) (*TodoModel, error)
	Update(id string, todo TodoModel) (*TodoModel, error)
	Delete(id string) error
}

// todoRepository implements the TodoRepository interface with in-memory storage
type todoRepository struct {
	todos []TodoModel
	mu    sync.RWMutex
}

// NewTodoRepository creates a new instance of TodoRepository
func NewTodoRepository() TodoRepository {
	return &todoRepository{
		todos: make([]TodoModel, 0),
	}
}

func (r *todoRepository) FindAll() ([]TodoModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Create a copy of the todos slice to prevent external modification
	todos := make([]TodoModel, len(r.todos))
	copy(todos, r.todos)
	return todos, nil
}

func (r *todoRepository) FindByID(id string) (*TodoModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, todo := range r.todos {
		if todo.Id == id {
			// Return a copy of the todo to prevent external modification
			todoCopy := todo
			return &todoCopy, nil
		}
	}
	return nil, errors.New("todo not found")
}

func (r *todoRepository) Create(todo TodoModel) (*TodoModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.todos = append(r.todos, todo)
	// Return a copy of the created todo
	todoCopy := todo
	return &todoCopy, nil
}

func (r *todoRepository) Update(id string, updatedTodo TodoModel) (*TodoModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, todo := range r.todos {
		if todo.Id == id {
			r.todos[i] = updatedTodo
			// Return a copy of the updated todo
			todoCopy := updatedTodo
			return &todoCopy, nil
		}
	}
	return nil, errors.New("todo not found")
}

func (r *todoRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, todo := range r.todos {
		if todo.Id == id {
			r.todos = append(r.todos[:i], r.todos[i+1:]...)
			return nil
		}
	}
	return errors.New("todo not found")
}
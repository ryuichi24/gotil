package todo

import (
	"github.com/google/uuid"
	"time"
)

// TodoService defines the interface for todo operations
type TodoService interface {
	GetAllTodos() ([]TodoModel, error)
	GetTodoByID(id string) (*TodoModel, error)
	CreateTodo(todo TodoModel) (*TodoModel, error)
	UpdateTodo(id string, todo TodoModel) (*TodoModel, error)
	DeleteTodo(id string) error
}

// todoService implements the TodoService interface
type todoService struct {
	repo TodoRepository
}

// NewTodoService creates a new instance of TodoService
func NewTodoService() TodoService {
	return &todoService{
		repo: NewTodoRepository(),
	}
}

func (s *todoService) GetAllTodos() ([]TodoModel, error) {
	return s.repo.FindAll()
}

func (s *todoService) GetTodoByID(id string) (*TodoModel, error) {
	return s.repo.FindByID(id)
}

func (s *todoService) CreateTodo(todo TodoModel) (*TodoModel, error) {
	todo.Id = uuid.New().String()
	todo.CreatedAt = time.Now().Format(time.RFC3339)
	todo.UpdatedAt = todo.CreatedAt
	return s.repo.Create(todo)
}

func (s *todoService) UpdateTodo(id string, updatedTodo TodoModel) (*TodoModel, error) {
	existingTodo, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	updatedTodo.Id = id
	updatedTodo.CreatedAt = existingTodo.CreatedAt
	updatedTodo.UpdatedAt = time.Now().Format(time.RFC3339)
	return s.repo.Update(id, updatedTodo)
}

func (s *todoService) DeleteTodo(id string) error {
	return s.repo.Delete(id)
}

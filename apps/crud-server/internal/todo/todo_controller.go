package todo

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func TodoController(engine *gin.Engine) {
	// Create a new router group for the todo routes
	todoRouter := engine.Group("/todos")

	// Initialize the todo service
	todoService := NewTodoService()

	todoRouter.GET("/", func(c *gin.Context) {
		todos, err := todoService.GetAllTodos()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": todos})
	})

	todoRouter.POST("/", func(c *gin.Context) {
		var todo TodoModel
		if err := c.ShouldBindJSON(&todo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		createdTodo, err := todoService.CreateTodo(todo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, createdTodo)
	})

	todoRouter.GET("/:id", func(c *gin.Context) {
		id := c.Param("id")
		todo, err := todoService.GetTodoByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, todo)
	})

	todoRouter.PUT("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var todo TodoModel
		if err := c.ShouldBindJSON(&todo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updatedTodo, err := todoService.UpdateTodo(id, todo)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, updatedTodo)
	})

	todoRouter.DELETE("/:id", func(c *gin.Context) {
		id := c.Param("id")
		err := todoService.DeleteTodo(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})
}

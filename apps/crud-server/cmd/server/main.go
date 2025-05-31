package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/ryuichi24/go-crud-server/internal/todo"
)

func main() {
	router := gin.Default()

	todo.TodoController(router)

	err := router.Run(":8080")

	if err != nil {
		log.Fatal("Failed to start server: ", err)
	}

	fmt.Println("server started at port 8080...")
}

package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type M map[string]any

func health(c *gin.Context) {
	c.JSON(http.StatusOK, M{"msg": "alive!"})
}

func main() {
	r := gin.Default()

	r.GET("/health", health)

	r.Run()

	fmt.Println("Running server")
}

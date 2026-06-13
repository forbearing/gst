package middleware

import (
	"math/rand/v2"
	"time"

	"github.com/forbearing/gst/middleware"
	"github.com/gin-gonic/gin"
)

func init() {
	middleware.Register(Middleware1, Middleware2, Middleware3)
}

func Middleware1(c *gin.Context) {
	n := rand.IntN(1000)

	time.Sleep(time.Duration(n) * time.Millisecond)
}

func Middleware2(c *gin.Context) {
	n := rand.IntN(1000)

	time.Sleep(time.Duration(n) * time.Millisecond)
}

func Middleware3(c *gin.Context) {
	n := rand.IntN(1000)

	time.Sleep(time.Duration(n) * time.Millisecond)
}

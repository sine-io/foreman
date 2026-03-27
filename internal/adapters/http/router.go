package http

import (
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/gin-gonic/gin"
)

func NewRouter(app App) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(gin.Recovery())

	handlers := NewBoardHandlers(app)
	router.GET("/board", func(c *gin.Context) {
		c.File(filepath.Join(boardAssetDir(), "index.html"))
	})
	router.StaticFS("/board/assets", http.Dir(boardAssetDir()))
	router.GET("/board/modules", handlers.ModuleBoard)
	router.GET("/board/tasks", handlers.TaskBoard)
	router.GET("/board/approvals", handlers.ApprovalQueue)
	router.GET("/board/runs/:id", handlers.RunDetail)
	router.POST("/board/tasks/:id/approve", handlers.ApproveTask)
	router.POST("/board/tasks/:id/retry", handlers.RetryTask)
	router.POST("/board/tasks/:id/cancel", handlers.CancelTask)
	router.POST("/board/tasks/:id/reprioritize", handlers.ReprioritizeTask)
	router.POST("/gateways/openclaw/command", handlers.OpenClawCommand)

	return router
}

func boardAssetDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "web", "board")
}

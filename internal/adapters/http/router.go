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
	router.GET("/board/approvals/workbench", func(c *gin.Context) {
		c.File(filepath.Join(boardAssetDir(), "approval-workbench.html"))
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

	if managerApp, ok := app.(ManagerApp); ok {
		managerHandlers := NewManagerHandlers(managerApp)
		router.POST("/api/manager/commands", managerHandlers.ManagerCommand)
		router.GET("/api/manager/tasks/:id", managerHandlers.ManagerTaskStatus)
		router.GET("/api/manager/projects/:id/board", managerHandlers.ManagerBoardSnapshot)
		router.GET("/api/manager/projects/:id/approvals", managerHandlers.ManagerApprovalWorkbenchQueue)
		router.GET("/api/manager/approvals/:id", managerHandlers.ManagerApprovalWorkbenchDetail)
		router.POST("/api/manager/approvals/:id/approve", managerHandlers.ApproveApproval)
		router.POST("/api/manager/approvals/:id/reject", managerHandlers.RejectApproval)
		router.POST("/api/manager/approvals/:id/retry-dispatch", managerHandlers.RetryApprovalDispatch)
	}

	return router
}

func boardAssetDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "web", "board")
}

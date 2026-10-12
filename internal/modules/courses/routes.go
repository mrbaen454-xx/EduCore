package courses

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all course and prerequisite endpoints to the router group.
func RegisterRoutes(rg *gin.RouterGroup, handler *Handler, authMiddleware, adminStaffMiddleware gin.HandlerFunc) {
	group := rg.Group("/courses")
	{
		// Read operations: authenticated users
		group.GET("", authMiddleware, handler.List)
		group.GET("/:id", authMiddleware, handler.GetByID)

		// Mutation operations: admin and akademik staff only
		group.POST("", authMiddleware, adminStaffMiddleware, handler.Create)
		group.PATCH("/:id", authMiddleware, adminStaffMiddleware, handler.Update)
		group.PATCH("/:id/status", authMiddleware, adminStaffMiddleware, handler.UpdateStatus)

		// Prerequisite operations
		group.POST("/:id/prerequisites", authMiddleware, adminStaffMiddleware, handler.AddPrerequisite)
		group.DELETE("/:id/prerequisites/:prerequisiteId", authMiddleware, adminStaffMiddleware, handler.RemovePrerequisite)
	}
}

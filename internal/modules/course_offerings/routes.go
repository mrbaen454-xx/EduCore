package course_offerings

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware, adminStaffMiddleware gin.HandlerFunc) {
	group := rg.Group("/course-offerings", authMiddleware)
	{
		// Read endpoints: authenticated users
		group.GET("", h.List)
		group.GET("/:id", h.GetByID)

		// Mutation endpoints: admin and staff (akademik)
		group.POST("", adminStaffMiddleware, h.Create)
		group.PATCH("/:id/status", adminStaffMiddleware, h.UpdateStatus)
	}
}

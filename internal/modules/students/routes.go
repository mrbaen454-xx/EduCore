package students

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all student endpoints onto the Gin router group.
func RegisterRoutes(router *gin.RouterGroup, handler *Handler, authMiddleware, adminStaffMiddleware gin.HandlerFunc) {
	group := router.Group("/students")
	group.Use(authMiddleware)
	{
		// Read operations: Accessible by all authenticated roles (admin, akademik, dosen, mahasiswa)
		group.GET("", handler.List)
		group.GET("/:id", handler.GetByID)

		// Mutation operations: Strictly restricted to admin and akademik roles
		mutations := group.Group("")
		mutations.Use(adminStaffMiddleware)
		{
			mutations.POST("", handler.Create)
			mutations.PATCH("/:id", handler.Update)
			mutations.PATCH("/:id/status", handler.UpdateStatus)
		}
	}
}

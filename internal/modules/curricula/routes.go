package curricula

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all curriculum and curriculum courses endpoints.
func RegisterRoutes(rg *gin.RouterGroup, handler *Handler, authMiddleware, adminStaffMiddleware gin.HandlerFunc) {
	group := rg.Group("/curricula")
	{
		// Curriculum Read operations
		group.GET("", authMiddleware, handler.List)
		group.GET("/:id", authMiddleware, handler.GetByID)

		// Curriculum Mutation operations
		group.POST("", authMiddleware, adminStaffMiddleware, handler.Create)
		group.PATCH("/:id", authMiddleware, adminStaffMiddleware, handler.Update)
		group.PATCH("/:id/status", authMiddleware, adminStaffMiddleware, handler.UpdateStatus)

		// Curriculum Courses operations
		coursesGroup := group.Group("/:id/courses")
		{
			coursesGroup.GET("", authMiddleware, handler.ListCourses)
			coursesGroup.GET("/:courseId", authMiddleware, handler.GetCourse)
			coursesGroup.POST("", authMiddleware, adminStaffMiddleware, handler.AddCourse)
			coursesGroup.PATCH("/:courseId", authMiddleware, adminStaffMiddleware, handler.UpdateCourse)
			coursesGroup.DELETE("/:courseId", authMiddleware, adminStaffMiddleware, handler.RemoveCourse)
		}
	}
}

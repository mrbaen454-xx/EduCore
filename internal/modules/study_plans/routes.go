package study_plans

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers student and administrative study plan / enrollment routes.
func RegisterRoutes(
	rg *gin.RouterGroup,
	h *Handler,
	authMiddleware gin.HandlerFunc,
	studentMiddleware gin.HandlerFunc,
	adminStaffMiddleware gin.HandlerFunc,
) {
	// 1. Student routes (/api/v1/student/...)
	studentGroup := rg.Group("/student")
	studentGroup.Use(authMiddleware, studentMiddleware)
	{
		studentGroup.GET("/course-offerings", h.GetStudentAvailableClasses)
		studentGroup.GET("/study-plans", h.GetMyStudyPlans)
		studentGroup.POST("/study-plans", h.CreateMyStudyPlan)
		studentGroup.GET("/study-plans/:id", h.GetMyStudyPlanDetail)
		studentGroup.POST("/study-plans/:id/enrollments", h.EnrollClassInMyStudyPlan)
		studentGroup.DELETE("/study-plans/:id/enrollments/:enrollmentId", h.DropEnrollmentFromMyStudyPlan)
	}

	// 2. Administrative routes (/api/v1/study-plans/...)
	adminGroup := rg.Group("/study-plans")
	adminGroup.Use(authMiddleware, adminStaffMiddleware)
	{
		adminGroup.GET("", h.ListStudyPlans)
		adminGroup.POST("", h.CreateStudyPlan)
		adminGroup.GET("/:id", h.GetStudyPlanDetail)
		adminGroup.POST("/:id/enrollments", h.EnrollClass)
		adminGroup.DELETE("/:id/enrollments/:enrollmentId", h.DropEnrollment)
	}
}

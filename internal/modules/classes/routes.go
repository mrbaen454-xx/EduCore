package classes

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware, adminStaffMiddleware gin.HandlerFunc) {
	group := rg.Group("/classes", authMiddleware)
	{
		// Read endpoints: authenticated users
		group.GET("", h.ListClasses)
		group.GET("/:id", h.GetClassByID)
		group.GET("/:id/lecturers", h.GetClassLecturers)
		group.GET("/:id/schedules", h.GetClassSchedules)

		// Mutation endpoints: admin and staff (akademik)
		group.POST("", adminStaffMiddleware, h.CreateClass)
		group.PATCH("/:id", adminStaffMiddleware, h.UpdateClass)
		group.PATCH("/:id/status", adminStaffMiddleware, h.UpdateClassStatus)

		// Lecturers mutation
		group.POST("/:id/lecturers", adminStaffMiddleware, h.AssignLecturer)
		group.DELETE("/:id/lecturers/:lecturerId", adminStaffMiddleware, h.RemoveLecturer)

		// Schedules mutation
		group.POST("/:id/schedules", adminStaffMiddleware, h.CreateSchedule)
		group.PATCH("/:id/schedules/:scheduleId", adminStaffMiddleware, h.UpdateSchedule)
		group.DELETE("/:id/schedules/:scheduleId", adminStaffMiddleware, h.DeleteSchedule)
	}
}

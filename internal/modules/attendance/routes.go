package attendance

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all HTTP routes for class sessions and student attendance.
func RegisterRoutes(
	rg *gin.RouterGroup,
	h *Handler,
	authMiddleware gin.HandlerFunc,
	academicUserMiddleware gin.HandlerFunc, // dosen, admin, akademik, mahasiswa
	teachingStaffMiddleware gin.HandlerFunc, // dosen, admin, akademik
	studentMiddleware gin.HandlerFunc, // mahasiswa
) {
	// 1. Student self-service routes (/api/v1/student/attendance)
	studentGroup := rg.Group("/student/attendance")
	studentGroup.Use(authMiddleware, studentMiddleware)
	{
		studentGroup.GET("", h.GetMyAttendanceHistory)
		studentGroup.GET("/classes/:id", h.GetMyClassAttendance)
	}

	// 2. Class sessions and class attendance summary (/api/v1/classes/:id/...)
	classesGroup := rg.Group("/classes/:id")
	classesGroup.Use(authMiddleware)
	{
		// Accessible to all academic users (lecturer, staff, enrolled students)
		classesGroup.GET("/sessions", academicUserMiddleware, h.ListSessionsByClass)

		// Teaching staff & admin operations
		classesGroup.POST("/sessions", teachingStaffMiddleware, h.CreateSession)
		classesGroup.GET("/attendance/summary", teachingStaffMiddleware, h.GetClassAttendanceSummary)
	}

	// 3. Direct session and attendance operations (/api/v1/sessions/:id/...)
	sessionsGroup := rg.Group("/sessions/:id")
	sessionsGroup.Use(authMiddleware)
	{
		// Detail accessible to academic users (lecturer, staff, enrolled students)
		sessionsGroup.GET("", academicUserMiddleware, h.GetSessionDetail)

		// Session lifecycle
		sessionsGroup.PATCH("", teachingStaffMiddleware, h.UpdateSession)
		sessionsGroup.POST("/close", teachingStaffMiddleware, h.CloseSession)

		// Attendance operations
		sessionsGroup.GET("/students", teachingStaffMiddleware, h.GetSessionRoster)
		sessionsGroup.GET("/attendance", teachingStaffMiddleware, h.GetSessionAttendance)
		sessionsGroup.POST("/attendance", teachingStaffMiddleware, h.RecordBatchAttendance)
		sessionsGroup.PATCH("/attendance/:attendanceId", teachingStaffMiddleware, h.UpdateAttendance)
	}
}

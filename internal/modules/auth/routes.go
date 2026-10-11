package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the authentication and user profile routes onto the Gin router group.
func RegisterRoutes(
	apiV1 *gin.RouterGroup,
	handler *Handler,
	authMiddleware gin.HandlerFunc,
	adminMiddleware gin.HandlerFunc,
) {
	authGroup := apiV1.Group("/auth")
	{
		authGroup.POST("/login", handler.Login)
		authGroup.POST("/refresh", handler.RefreshToken)
		authGroup.POST("/logout", handler.Logout)
	}

	usersGroup := apiV1.Group("/users")
	{
		usersGroup.GET("/me", authMiddleware, handler.GetProfile)
	}

	// Verification endpoint to test role-based access control (RBAC)
	testGroup := apiV1.Group("/test")
	{
		testGroup.GET("/admin-only", authMiddleware, adminMiddleware, handler.TestAdminOnly)
	}
}

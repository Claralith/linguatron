package routes

import (
	"webproject/database"
	"webproject/routes/api"

	"github.com/gin-gonic/gin"
)

func RegisterAll(r *gin.Engine, gormDB *database.GormDB) {

	api.RegisterDecksRoutes(r, gormDB)
	api.RegisterReviewRoutes(r, gormDB)
	api.RegisterSetupRoutes(r, gormDB)
	api.RegisterLearningRoutes(r, gormDB)
}

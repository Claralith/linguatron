package routes

import (
	"webproject/database"
	routestempl "webproject/routes/templ"

	"github.com/gin-gonic/gin"
)

func RegisterAll(r *gin.Engine, gormDB *database.GormDB) {

	routestempl.RegisterTemplSetupRoutes(r, gormDB)
	routestempl.RegisterTemplLearningRoutes(r, gormDB)
	routestempl.RegisterTemplReviewRoutes(r, gormDB)
	routestempl.RegisterTemplDecksRoutes(r, gormDB)
}

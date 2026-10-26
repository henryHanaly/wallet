package router

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"api/internal/controller"
	"api/internal/middleware"
)

func New(wallet *controller.WalletController) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Logger(log.StandardLogger()), gin.Recovery())

	r.GET("/ping", controller.Ping)
	r.POST("/create", wallet.CreateUser)
	r.GET("/getBalance/:username", wallet.GetBalance)
	r.POST("/transfer", wallet.Transfer)

	return r
}

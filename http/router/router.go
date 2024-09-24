package router

import (
	"Gwen/global"
	"Gwen/http/controller/web"
	"github.com/gin-gonic/gin"
	"net/http"
)

func WebInit(g *gin.Engine) {
	i := &web.Index{}
	g.GET("/", i.Index)
	g.GET("/webclient-config/index.js", i.ConfigJs)
	g.StaticFS("/webclient", http.Dir(global.Config.Gin.ResourcesPath+"/web"))
	g.StaticFS("/_admin", http.Dir(global.Config.Gin.ResourcesPath+"/admin"))
}

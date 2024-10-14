package admin

import (
	"Gwen/global"
	"Gwen/http/request/admin"
	"Gwen/http/response"
	adResp "Gwen/http/response/admin"
	"Gwen/model"
	"Gwen/service"
	"fmt"
	"github.com/gin-gonic/gin"
)

type Login struct {
}

// Login 登录
// @Tags 登录
// @Summary 登录
// @Description 登录
// @Accept  json
// @Produce  json
// @Param body body admin.Login true "登录信息"
// @Success 200 {object} response.Response{data=adResp.LoginPayload}
// @Failure 500 {object} response.Response
// @Router /admin/login [post]
// @Security token
func (ct *Login) Login(c *gin.Context) {
	f := &admin.Login{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		global.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), c.ClientIP()))
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		global.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), c.ClientIP()))
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.UserService.InfoByUsernamePassword(f.Username, f.Password)

	if u.Id == 0 {
		global.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "UsernameOrPasswordError", c.RemoteIP(), c.ClientIP()))
		response.Fail(c, 101, response.TranslateMsg(c, "UsernameOrPasswordError"))
		return
	}

	ut := service.AllService.UserService.Login(u, &model.LoginLog{
		UserId:   u.Id,
		Client:   "webadmin",
		Uuid:     "",
		Ip:       c.ClientIP(),
		Type:     "account",
		Platform: f.Platform,
	})

	response.Success(c, &adResp.LoginPayload{
		Token:      ut.Token,
		Username:   u.Username,
		RouteNames: service.AllService.UserService.RouteNames(u),
		Nickname:   u.Nickname,
	})
}

// Logout 登出
// @Tags 登录
// @Summary 登出
// @Description 登出
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/logout [post]
func (ct *Login) Logout(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	token, ok := c.Get("token")
	if ok {
		service.AllService.UserService.Logout(u, token.(string))
	}
	response.Success(c, nil)
}

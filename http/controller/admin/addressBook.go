package admin

import (
	"Gwen/global"
	"Gwen/http/request/admin"
	"Gwen/http/response"
	"Gwen/model"
	"Gwen/service"
	_ "encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strconv"
)

type AddressBook struct {
}

// Detail 地址簿
// @Tags 地址簿
// @Summary 地址簿详情
// @Description 地址簿详情
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.AddressBook}
// @Failure 500 {object} response.Response
// @Router /admin/address_book/detail/{id} [get]
// @Security token
func (ct *AddressBook) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	t := service.AllService.AddressBookService.InfoByRowId(uint(iid))
	u := service.AllService.UserService.CurUser(c)
	if !service.AllService.UserService.IsAdmin(u) && t.UserId != u.Id {
		response.Fail(c, 101, response.TranslateMsg(c, "NoAccess"))
		return
	}
	if t.RowId > 0 {
		response.Success(c, t)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
	return
}

// Create 创建地址簿
// @Tags 地址簿
// @Summary 创建地址簿
// @Description 创建地址簿
// @Accept  json
// @Produce  json
// @Param body body admin.AddressBookForm true "地址簿信息"
// @Success 200 {object} response.Response{data=model.AddressBook}
// @Failure 500 {object} response.Response
// @Router /admin/address_book/create [post]
// @Security token
func (ct *AddressBook) Create(c *gin.Context) {
	f := &admin.AddressBookForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	t := f.ToAddressBook()
	u := service.AllService.UserService.CurUser(c)
	if !service.AllService.UserService.IsAdmin(u) || t.UserId == 0 {
		t.UserId = u.Id
	}
	ex := service.AllService.AddressBookService.InfoByUserIdAndId(t.UserId, t.Id)
	if ex.RowId > 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemExist"))
		return
	}

	err := service.AllService.AddressBookService.Create(t)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, u)
}

// BatchCreate 批量创建地址簿
// @Tags 地址簿
// @Summary 批量创建地址簿
// @Description 批量创建地址簿
// @Accept  json
// @Produce  json
// @Param body body admin.AddressBookForm true "地址簿信息"
// @Success 200 {object} response.Response{data=model.AddressBook}
// @Failure 500 {object} response.Response
// @Router /admin/address_book/create [post]
// @Security token
func (ct *AddressBook) BatchCreate(c *gin.Context) {
	f := &admin.AddressBookForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}

	//创建标签
	for _, fu := range f.UserIds {
		if fu == 0 {
			continue
		}
		for _, ft := range f.Tags {
			exTag := service.AllService.TagService.InfoByUserIdAndName(fu, ft)
			if exTag.Id == 0 {
				service.AllService.TagService.Create(&model.Tag{
					UserId: fu,
					Name:   ft,
				})
			}
		}
	}
	ts := f.ToAddressBooks()
	for _, t := range ts {
		if t.UserId == 0 {
			continue
		}
		ex := service.AllService.AddressBookService.InfoByUserIdAndId(t.UserId, t.Id)
		if ex.RowId == 0 {
			service.AllService.AddressBookService.Create(t)
		}
	}

	response.Success(c, nil)
}

// List 列表
// @Tags 地址簿
// @Summary 地址簿列表
// @Description 地址簿列表
// @Accept  json
// @Produce  json
// @Param page query int false "页码"
// @Param page_size query int false "页大小"
// @Param user_id query int false "用户id"
// @Param is_my query int false "是否是我的"
// @Success 200 {object} response.Response{data=model.AddressBookList}
// @Failure 500 {object} response.Response
// @Router /admin/address_book/list [get]
// @Security token
func (ct *AddressBook) List(c *gin.Context) {
	query := &admin.AddressBookQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := service.AllService.UserService.CurUser(c)
	if !service.AllService.UserService.IsAdmin(u) || query.IsMy == 1 {
		query.UserId = int(u.Id)
	}
	res := service.AllService.AddressBookService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
		if query.Id != "" {
			tx.Where("id like ?", "%"+query.Id+"%")
		}
		if query.UserId > 0 {
			tx.Where("user_id = ?", query.UserId)
		}
		if query.Username != "" {
			tx.Where("username like ?", "%"+query.Username+"%")
		}
		if query.Hostname != "" {
			tx.Where("hostname like ?", "%"+query.Hostname+"%")
		}
	})
	response.Success(c, res)
}

// Update 编辑
// @Tags 地址簿
// @Summary 地址簿编辑
// @Description 地址簿编辑
// @Accept  json
// @Produce  json
// @Param body body admin.AddressBookForm true "地址簿信息"
// @Success 200 {object} response.Response{data=model.AddressBook}
// @Failure 500 {object} response.Response
// @Router /admin/address_book/update [post]
// @Security token
func (ct *AddressBook) Update(c *gin.Context) {
	f := &admin.AddressBookForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	if f.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	t := f.ToAddressBook()
	u := service.AllService.UserService.CurUser(c)
	if !service.AllService.UserService.IsAdmin(u) && t.UserId != u.Id {
		response.Fail(c, 101, response.TranslateMsg(c, "NoAccess"))
		return
	}
	err := service.AllService.AddressBookService.Update(t)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete 删除
// @Tags 地址簿
// @Summary 地址簿删除
// @Description 地址簿删除
// @Accept  json
// @Produce  json
// @Param body body admin.AddressBookForm true "地址簿信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/address_book/delete [post]
// @Security token
func (ct *AddressBook) Delete(c *gin.Context) {
	f := &admin.AddressBookForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	id := f.RowId
	errList := global.Validator.ValidVar(c, id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	t := service.AllService.AddressBookService.InfoByRowId(f.RowId)
	u := service.AllService.UserService.CurUser(c)
	if !service.AllService.UserService.IsAdmin(u) && t.UserId != u.Id {
		response.Fail(c, 101, response.TranslateMsg(c, "NoAccess"))
		return
	}
	if u.Id > 0 {
		err := service.AllService.AddressBookService.Delete(t)
		if err == nil {
			response.Success(c, nil)
			return
		}
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// ShareByWebClient
// @Tags 地址簿
// @Summary 地址簿分享
// @Description 地址簿分享
// @Accept  json
// @Produce  json
// @Param body body admin.ShareByWebClientForm true "地址簿信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/address_book/share [post]
// @Security token
func (ct *AddressBook) ShareByWebClient(c *gin.Context) {
	f := &admin.ShareByWebClientForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}

	u := service.AllService.UserService.CurUser(c)
	ab := service.AllService.AddressBookService.InfoByUserIdAndId(u.Id, f.Id)
	if ab.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	m := f.ToShareRecord()
	m.UserId = u.Id
	err := service.AllService.AddressBookService.ShareByWebClient(m)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, &gin.H{
		"share_token": m.ShareToken,
	})
}

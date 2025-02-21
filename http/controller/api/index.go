package api

import (
	"github.com/gin-gonic/gin"
	requstform "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"net/http"
	"os"
	"time"
	"sync"
)

type Index struct {
}

// 缓存变量
var peerCache sync.Map

// 定时将缓存中的数据更新到数据库的间隔
//var updateInterval = 1 * time.Hour

// Index 首页
// @Tags 首页
// @Summary 首页
// @Description 首页
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router / [get]
func (i *Index) Index(c *gin.Context) {
	response.Success(
		c,
		"Hello Gwen",
	)
}

// UpdateCacheToDB 遍历缓存并将数据更新到数据库
func UpdateCacheToDB(wg *sync.WaitGroup) {
	defer wg.Done() // 当函数结束时，通知 WaitGroup 完成任务
	peerCache.Range(func(key, value interface{}) bool {
		peer := value.(*model.Peer) // 从缓存中取出 model.Peer
		service.AllService.PeerService.Update(peer) // 更新到数据库
		peerCache.Delete(key) // 更新后从缓存中删除
		return true
	})
}

//func init() {
	// 定时将缓存的数据写入数据库
//	go func() {
//		ticker := time.NewTicker(updateInterval)
//		defer ticker.Stop()
//		for {
//			<-ticker.C
			// 调用缓存更新函数
//			UpdateCacheToDB()
//		}
//	}()
//}

// Heartbeat 心跳
// @Tags 首页
// @Summary 心跳
// @Description 心跳
// @Accept  json
// @Produce  json
// @Success 200 {object} nil
// @Failure 500 {object} response.Response
// @Router /heartbeat [post]
func (i *Index) Heartbeat(c *gin.Context) {
	info := &requstform.PeerInfoInHeartbeat{}
	err := c.ShouldBindJSON(info)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	if info.Uuid == "" {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	// 先从缓存中查找对应的 `peer` 数据
	peerInterface, ok := peerCache.Load(info.Uuid)
	var peer *model.Peer

	// 如果缓存中不存在数据，从数据库中查找
	if !ok {
		peer = service.AllService.PeerService.FindByUuid(info.Uuid)
		if peer == nil || peer.RowId == 0 {
			// 如果数据库中也找不到，返回空响应
			c.JSON(http.StatusOK, gin.H{})
			return
		}
	} else {
		// 如果缓存中存在数据，转换类型
		peer = peerInterface.(*model.Peer)
	}

	// 更新 `LastOnlineTime` 和 `LastOnlineIp` 字段
	peer.LastOnlineTime = time.Now().Unix()
	peer.LastOnlineIp = c.ClientIP()

	// 将更新后的数据重新存入缓存
	peerCache.Store(info.Uuid, peer)
	c.JSON(http.StatusOK, gin.H{})
}

// Version 版本
// @Tags 首页
// @Summary 版本
// @Description 版本
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /version [get]
func (i *Index) Version(c *gin.Context) {
	//读取resources/version文件
	v, err := os.ReadFile("resources/version")
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(
		c,
		string(v),
	)
}

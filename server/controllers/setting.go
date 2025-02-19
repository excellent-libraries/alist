package controllers

import (
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/sign"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
	"strconv"
)

func ResetToken(c *gin.Context) {
	token := random.Token()
	item := model.SettingItem{Key: "token", Value: token, Type: conf.TypeString, Group: model.SINGLE, Flag: model.PRIVATE}
	if err := db.SaveSettingItem(item); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	sign.Instance()
	common.SuccessResp(c, token)
}

func GetSetting(c *gin.Context) {
	key := c.Query("key")
	item, err := db.GetSettingItemByKey(key)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	common.SuccessResp(c, item)
}

func SaveSettings(c *gin.Context) {
	var req []model.SettingItem
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := db.SaveSettingItems(req); err != nil {
		common.ErrorResp(c, err, 500)
	} else {
		common.SuccessResp(c)
	}
}

func ListSettings(c *gin.Context) {
	groupStr := c.Query("group")
	var settings []model.SettingItem
	var err error
	if groupStr == "" {
		settings, err = db.GetSettingItems()
	} else {
		group, err := strconv.Atoi(groupStr)
		if err == nil {
			settings, err = db.GetSettingItemsByGroup(group)
		}
	}
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	common.SuccessResp(c, settings)
}

func DeleteSetting(c *gin.Context) {
	key := c.Query("key")
	if err := db.DeleteSettingItemByKey(key); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}

func PublicSettings(c *gin.Context) {
	common.SuccessResp(c, db.GetPublicSettingsMap())
}

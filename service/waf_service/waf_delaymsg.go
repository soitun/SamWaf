package waf_service

import (
	"SamWaf/global"
	"SamWaf/model"
	uuid "github.com/satori/go.uuid"
	"time"
)

type WafDelayMsgService struct{}

var WafDelayMsgServiceApp = new(WafDelayMsgService)

func (receiver *WafDelayMsgService) Add(DelayType, DelayTile, DelayContent string) error {
	var bean = &model.DelayMsg{
		UserCode:     global.GWAF_USER_CODE,
		TenantId:     global.GWAF_TENANT_ID,
		Id:           uuid.NewV4().String(),
		DelayType:    DelayType,
		DelayTile:    DelayTile,
		DelayContent: DelayContent,
		CreateTime:   time.Now(),
	}
	global.GWAF_LOCAL_DB.Create(bean)
	return nil
}
func (receiver *WafDelayMsgService) GetAllList() ([]model.DelayMsg, int64, error) {
	var ipWhites []model.DelayMsg
	var total int64 = 0
	global.GWAF_LOCAL_DB.Find(&ipWhites)
	global.GWAF_LOCAL_DB.Model(&model.DelayMsg{}).Count(&total)
	return ipWhites, total, nil
}
func (receiver *WafDelayMsgService) DelApi(id string) error {
	var bean model.DelayMsg
	err := global.GWAF_LOCAL_DB.Where("id = ?", id).First(&bean).Error
	if err != nil {
		return err
	}
	err = global.GWAF_LOCAL_DB.Where("id = ?", id).Delete(model.DelayMsg{}).Error
	return err
}
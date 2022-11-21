package api

import (
	"SamWaf/model/common/response"
	"SamWaf/model/request"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WafLdpUrlApi struct {
}

func (w *WafLdpUrlApi) AddApi(c *gin.Context) {
	var req request.WafLdpUrlAddReq
	err := c.ShouldBind(&req)
	if err == nil {
		err = wafLdpUrlService.CheckIsExistApi(req)
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			err = wafLdpUrlService.AddApi(req)
			if err == nil {

				response.OkWithMessage("添加成功", c)
			} else {

				response.FailWithMessage("添加失败", c)
			}
			return
		} else {
			response.FailWithMessage("当前网站的Url已经存在", c)
			return
		}

	} else {
		response.FailWithMessage("解析失败", c)
	}
}
func (w *WafLdpUrlApi) GetDetailApi(c *gin.Context) {
	var req request.WafLdpUrlDetailReq
	err := c.ShouldBind(&req)
	if err == nil {
		bean := wafLdpUrlService.GetDetailApi(req)
		response.OkWithDetailed(bean, "获取成功", c)
	} else {
		response.FailWithMessage("解析失败", c)
	}
}
func (w *WafLdpUrlApi) GetListApi(c *gin.Context) {
	var req request.WafLdpUrlSearchReq
	err := c.ShouldBind(&req)
	if err == nil {
		beans, total, _ := wafLdpUrlService.GetListApi(req)
		response.OkWithDetailed(response.PageResult{
			List:      beans,
			Total:     total,
			PageIndex: req.PageIndex,
			PageSize:  req.PageSize,
		}, "获取成功", c)
	} else {
		response.FailWithMessage("解析失败", c)
	}
}
func (w *WafLdpUrlApi) DelLdpUrlApi(c *gin.Context) {
	var req request.WafLdpUrlDelReq
	err := c.ShouldBind(&req)
	if err == nil {
		err = wafLdpUrlService.DelApi(req)
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithMessage("请检测参数", c)
		} else if err != nil {
			response.FailWithMessage("发生错误", c)
		} else {
			response.FailWithMessage("删除成功", c)
		}

	} else {
		response.FailWithMessage("解析失败", c)
	}
}

func (w *WafLdpUrlApi) ModifyLdpUrlApi(c *gin.Context) {
	var req request.WafLdpUrlEditReq
	err := c.ShouldBind(&req)
	if err == nil {
		err = wafLdpUrlService.ModifyApi(req)
		if err != nil {
			response.FailWithMessage("编辑发生错误", c)
		} else {
			response.OkWithMessage("编辑成功", c)
		}

	} else {
		response.FailWithMessage("解析失败", c)
	}
}
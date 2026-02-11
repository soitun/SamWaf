package api

import (
	"SamWaf/global"
	"SamWaf/iplocation"
	"SamWaf/model/common/response"
	"SamWaf/utils"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type WafIPLocationApi struct {
}

// GetIPDBStatusApi 获取 IP 数据库状态
func (w *WafIPLocationApi) GetIPDBStatusApi(c *gin.Context) {
	if global.GIPLOCATION_MANAGER == nil {
		response.FailWithMessage("IP 数据库管理器未初始化", c)
		return
	}

	status := global.GIPLOCATION_MANAGER.GetStatus()

	// 获取文件的实际创建时间
	dataDir := filepath.Join(utils.GetCurrentDir(), "data")

	// 获取 IPv4 文件创建时间
	var ipv4FilePath string
	if status.IPv4Source == "ip2region" {
		ipv4FilePath = filepath.Join(dataDir, "ip2region.xdb")
	} else if status.IPv4Source == "geolite2" {
		ipv4FilePath = filepath.Join(dataDir, "GeoLite2-Country.mmdb")
	}
	if ipv4FilePath != "" {
		if fileInfo, err := os.Stat(ipv4FilePath); err == nil {
			status.IPv4CreateTime = fileInfo.ModTime().Format("2006-01-02 15:04:05")
		}
	}

	// 获取 IPv6 文件创建时间
	var ipv6FilePath string
	if status.IPv6Source == "ip2region" {
		ipv6FilePath = filepath.Join(dataDir, "ip2region_v6.xdb")
	} else if status.IPv6Source == "geolite2" {
		ipv6FilePath = filepath.Join(dataDir, "GeoLite2-Country.mmdb")
	}
	if ipv6FilePath != "" {
		if fileInfo, err := os.Stat(ipv6FilePath); err == nil {
			status.IPv6CreateTime = fileInfo.ModTime().Format("2006-01-02 15:04:05")
		}
	}

	response.OkWithDetailed(status, "获取成功", c)
}

// UploadIPDBFileApi 上传 IP 数据库文件
func (w *WafIPLocationApi) UploadIPDBFileApi(c *gin.Context) {
	// 获取文件类型 (ipv4/ipv6)
	ipType := c.PostForm("type")
	if ipType != "ipv4" && ipType != "ipv6" {
		response.FailWithMessage("无效的类型参数，必须是 ipv4 或 ipv6", c)
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithMessage("文件上传失败: "+err.Error(), c)
		return
	}

	// 检查文件扩展名
	ext := filepath.Ext(file.Filename)
	if ext != ".xdb" && ext != ".mmdb" {
		response.FailWithMessage("不支持的文件类型，仅支持 .xdb 和 .mmdb 文件", c)
		return
	}

	// 确定保存路径
	var finalPath string
	dataDir := filepath.Join(utils.GetCurrentDir(), "data")

	// 确保 data 目录存在
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		err = os.MkdirAll(dataDir, 0755)
		if err != nil {
			response.FailWithMessage("创建数据目录失败: "+err.Error(), c)
			return
		}
	}

	if ipType == "ipv4" {
		if ext == ".xdb" {
			finalPath = filepath.Join(dataDir, "ip2region.xdb")
		} else {
			finalPath = filepath.Join(dataDir, "GeoLite2-Country.mmdb")
		}
	} else {
		if ext == ".xdb" {
			finalPath = filepath.Join(dataDir, "ip2region_v6.xdb")
		} else {
			finalPath = filepath.Join(dataDir, "GeoLite2-Country.mmdb")
		}
	}

	// 先保存到临时文件
	tempPath := finalPath + ".tmp"
	err = c.SaveUploadedFile(file, tempPath)
	if err != nil {
		response.FailWithMessage("保存临时文件失败: "+err.Error(), c)
		return
	}

	// 读取临时文件内容
	fileData, err := ioutil.ReadFile(tempPath)
	if err != nil {
		os.Remove(tempPath) // 清理临时文件
		response.FailWithMessage("读取文件失败: "+err.Error(), c)
		return
	}

	// 先热加载到内存，验证文件有效性
	if global.GIPLOCATION_MANAGER != nil {
		var reloadErr error

		if ipType == "ipv4" {
			if ext == ".xdb" {
				reloadErr = global.GIPLOCATION_MANAGER.LoadV4Ip2Region(fileData, iplocation.DBFormat(global.GCONFIG_IP_V4_FORMAT))
				if reloadErr == nil {
					global.GCONFIG_IP_V4_SOURCE = "ip2region"
				}
			} else {
				reloadErr = global.GIPLOCATION_MANAGER.LoadV4GeoLite2(fileData)
				if reloadErr == nil {
					global.GCONFIG_IP_V4_SOURCE = "geolite2"
				}
			}
		} else {
			if ext == ".xdb" {
				reloadErr = global.GIPLOCATION_MANAGER.LoadV6Ip2Region(fileData, iplocation.DBFormat(global.GCONFIG_IP_V6_FORMAT))
				if reloadErr == nil {
					global.GCONFIG_IP_V6_SOURCE = "ip2region"
				}
			} else {
				reloadErr = global.GIPLOCATION_MANAGER.LoadV6GeoLite2(fileData)
				if reloadErr == nil {
					global.GCONFIG_IP_V6_SOURCE = "geolite2"
				}
			}
		}

		if reloadErr != nil {
			os.Remove(tempPath) // 清理临时文件
			response.FailWithMessage("加载数据库失败: "+reloadErr.Error(), c)
			return
		}
	}

	// 热加载成功后，原子替换正式文件
	err = os.Rename(tempPath, finalPath)
	if err != nil {
		os.Remove(tempPath) // 清理临时文件
		response.FailWithMessage("替换文件失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("文件上传成功并已重新加载", c)
}

// ReloadIPDBApi 重新加载 IP 数据库
func (w *WafIPLocationApi) ReloadIPDBApi(c *gin.Context) {
	if global.GIPLOCATION_MANAGER == nil {
		response.FailWithMessage("IP 数据库管理器未初始化", c)
		return
	}

	dataDir := filepath.Join(utils.GetCurrentDir(), "data")

	// 重新加载 IPv4
	if global.GCONFIG_IP_V4_SOURCE == "ip2region" {
		ipv4Path := filepath.Join(dataDir, "ip2region.xdb")
		if _, err := os.Stat(ipv4Path); err == nil {
			data, err := ioutil.ReadFile(ipv4Path)
			if err == nil {
				err = global.GIPLOCATION_MANAGER.LoadV4Ip2Region(data, iplocation.DBFormat(global.GCONFIG_IP_V4_FORMAT))
				if err != nil {
					response.FailWithMessage("重新加载 IPv4 数据库失败: "+err.Error(), c)
					return
				}
			}
		}
	} else if global.GCONFIG_IP_V4_SOURCE == "geolite2" {
		ipv4Path := filepath.Join(dataDir, "GeoLite2-Country.mmdb")
		if _, err := os.Stat(ipv4Path); err == nil {
			data, err := ioutil.ReadFile(ipv4Path)
			if err == nil {
				err = global.GIPLOCATION_MANAGER.LoadV4GeoLite2(data)
				if err != nil {
					response.FailWithMessage("重新加载 IPv4 数据库失败: "+err.Error(), c)
					return
				}
			}
		}
	}

	// 重新加载 IPv6
	if global.GCONFIG_IP_V6_SOURCE == "ip2region" {
		ipv6Path := filepath.Join(dataDir, "ip2region_v6.xdb")
		if _, err := os.Stat(ipv6Path); err == nil {
			data, err := ioutil.ReadFile(ipv6Path)
			if err == nil {
				err = global.GIPLOCATION_MANAGER.LoadV6Ip2Region(data, iplocation.DBFormat(global.GCONFIG_IP_V6_FORMAT))
				if err != nil {
					response.FailWithMessage("重新加载 IPv6 数据库失败: "+err.Error(), c)
					return
				}
			}
		}
	} else if global.GCONFIG_IP_V6_SOURCE == "geolite2" {
		ipv6Path := filepath.Join(dataDir, "GeoLite2-Country.mmdb")
		if _, err := os.Stat(ipv6Path); err == nil {
			data, err := ioutil.ReadFile(ipv6Path)
			if err == nil {
				err = global.GIPLOCATION_MANAGER.LoadV6GeoLite2(data)
				if err != nil {
					response.FailWithMessage("重新加载 IPv6 数据库失败: "+err.Error(), c)
					return
				}
			}
		}
	}

	response.OkWithMessage("数据库重新加载成功", c)
}

// TestIPLookupApi 测试 IP 查询
func (w *WafIPLocationApi) TestIPLookupApi(c *gin.Context) {
	var req struct {
		IP string `json:"ip" binding:"required"`
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage("参数解析失败", c)
		return
	}

	if global.GIPLOCATION_MANAGER == nil {
		response.FailWithMessage("IP 数据库管理器未初始化", c)
		return
	}

	result := global.GIPLOCATION_MANAGER.Lookup(req.IP)

	resp := map[string]interface{}{
		"ip":       req.IP,
		"country":  result.Country,
		"province": result.Province,
		"city":     result.City,
		"isp":      result.ISP,
		"region":   result.Region,
		"district": result.District,
		"raw":      fmt.Sprintf("%v", result.ToSlice()),
	}

	response.OkWithDetailed(resp, "查询成功", c)
}

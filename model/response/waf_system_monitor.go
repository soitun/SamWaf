package response

// 系统监控响应结构体
type WafSystemMonitor struct {
	CPU    WafCPUInfo    `json:"cpu"`    // CPU信息
	Memory WafMemoryInfo `json:"memory"` // 内存信息
	Disk   []WafDiskInfo `json:"disk"`   // 磁盘信息
}

// CPU信息
type WafCPUInfo struct {
	ModelName    string  `json:"model_name"`    // CPU型号
	Cores        int32   `json:"cores"`         // 核心数
	UsagePercent float64 `json:"usage_percent"` // 使用率百分比
	PhysicalCnt  int     `json:"physical_cnt"`  // 物理核心数
	LogicalCnt   int     `json:"logical_cnt"`   // 逻辑核心数
}

// 内存信息
type WafMemoryInfo struct {
	Total        string  `json:"total"`         // 总内存
	Available    string  `json:"available"`     // 可用内存
	Used         string  `json:"used"`          // 已用内存
	UsagePercent float64 `json:"usage_percent"` // 使用率百分比
	JVMUsed      string  `json:"jvm_used"`      // JVM使用内存(这里用Go程序内存)
	JVMPercent   float64 `json:"jvm_percent"`   // JVM使用率
}

// 磁盘信息
type WafDiskInfo struct {
	FileSystem   string  `json:"file_system"`   // 文件系统
	MountPoint   string  `json:"mount_point"`   // 挂载点
	Total        string  `json:"total"`         // 总容量
	Available    string  `json:"available"`     // 可用容量
	Used         string  `json:"used"`          // 已用容量
	UsagePercent float64 `json:"usage_percent"` // 使用率百分比
}

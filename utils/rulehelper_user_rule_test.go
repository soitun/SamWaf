package utils

import (
	"SamWaf/innerbean"
	"SamWaf/model"
	"testing"
)

// TestUserRule_IPInRanges 测试用户原始规则的改写版本 - IPInRanges 方式
func TestUserRule_IPInRanges(t *testing.T) {
	ruleHelper := &RuleHelper{}
	ruleHelper.InitRuleEngine()

	// 用户改写后的规则 - 使用 IPInRanges（推荐方式）
	drls := `
rule R835f9bf09867473dbe873027241db107 "允许特定内网网段访问" salience 10 {
    when
        RF.IPInRanges(MF.SRC_IP, "172.16.0.0-172.20.255.254", "192.168.0.0-192.168.1.254") == true
    then
        Retract("R835f9bf09867473dbe873027241db107");
}`

	var ruleconfigs []model.Rules
	rule := model.Rules{
		HostCode:        "",
		RuleCode:        "R835f9bf09867473dbe873027241db107",
		RuleName:        "允许特定内网网段访问",
		RuleContent:     drls,
		RuleContentJSON: "",
		RuleVersionName: "1.0",
		RuleVersion:     1,
		IsPublicRule:    0,
		IsManualRule:    1,
		RuleStatus:      1,
	}
	ruleconfigs = append(ruleconfigs, rule)

	_, err := ruleHelper.LoadRules(ruleconfigs)
	if err != nil {
		t.Errorf("加载规则失败: %v", err)
		return
	}

	// 测试用例
	testCases := []struct {
		name     string
		ip       string
		expected bool // true 表示应该匹配规则（在范围内），false 表示不应该匹配
	}{
		{"172网段-起始IP", "172.16.0.0", true},
		{"172网段-中间IP", "172.18.100.50", true},
		{"172网段-结束IP", "172.20.255.254", true},
		{"172网段-超出范围", "172.21.0.0", false},
		{"192.168网段-起始IP", "192.168.0.0", true},
		{"192.168网段-中间IP", "192.168.0.100", true},
		{"192.168网段-结束IP", "192.168.1.254", true},
		{"192.168网段-超出范围", "192.168.2.0", false},
		{"外网IP", "8.8.8.8", false},
		{"本地IP", "127.0.0.1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			webLog := &innerbean.WebLog{
				SRC_IP: tc.ip,
			}

			ruleMatches, err := ruleHelper.Match("MF", webLog)
			if err != nil {
				t.Errorf("规则匹配错误: %v", err)
				return
			}

			matched := len(ruleMatches) > 0
			if matched != tc.expected {
				t.Errorf("IP %s: 期望匹配=%v, 实际匹配=%v", tc.ip, tc.expected, matched)
			} else {
				t.Logf("✓ IP %s: 匹配结果正确 (匹配=%v)", tc.ip, matched)
			}
		})
	}
}

// TestUserRule_IPInRange 测试用户原始规则的改写版本 - IPInRange 方式
func TestUserRule_IPInRange(t *testing.T) {
	ruleHelper := &RuleHelper{}
	ruleHelper.InitRuleEngine()

	// 用户改写后的规则 - 使用多个 IPInRange
	drls := `
rule R835f9bf09867473dbe873027241db107 "允许特定内网网段访问" salience 10 {
    when
        RF.IPInRange(MF.SRC_IP, "172.16.0.0", "172.20.255.254") == true ||
        RF.IPInRange(MF.SRC_IP, "192.168.0.0", "192.168.1.254") == true
    then
        Retract("R835f9bf09867473dbe873027241db107");
}`

	var ruleconfigs []model.Rules
	rule := model.Rules{
		HostCode:        "",
		RuleCode:        "R835f9bf09867473dbe873027241db107",
		RuleName:        "允许特定内网网段访问",
		RuleContent:     drls,
		RuleContentJSON: "",
		RuleVersionName: "1.0",
		RuleVersion:     1,
		IsPublicRule:    0,
		IsManualRule:    1,
		RuleStatus:      1,
	}
	ruleconfigs = append(ruleconfigs, rule)

	_, err := ruleHelper.LoadRules(ruleconfigs)
	if err != nil {
		t.Errorf("加载规则失败: %v", err)
		return
	}

	// 测试 172 网段内的 IP
	webLog := &innerbean.WebLog{
		SRC_IP: "172.18.0.1",
	}

	ruleMatches, err := ruleHelper.Match("MF", webLog)
	if err != nil {
		t.Errorf("规则匹配错误: %v", err)
		return
	}

	if len(ruleMatches) == 0 {
		t.Errorf("172.18.0.1 应该匹配规则")
	} else {
		t.Logf("✓ 规则匹配成功: %s", ruleMatches[0].RuleDescription)
	}

	// 测试外网 IP
	webLog2 := &innerbean.WebLog{
		SRC_IP: "8.8.8.8",
	}

	ruleMatches2, err := ruleHelper.Match("MF", webLog2)
	if err != nil {
		t.Errorf("规则匹配错误: %v", err)
		return
	}

	if len(ruleMatches2) > 0 {
		t.Errorf("8.8.8.8 不应该匹配规则")
	} else {
		t.Logf("✓ 外网IP正确未匹配")
	}
}

// TestUserRule_CIDR 测试用户规则 - CIDR 格式
func TestUserRule_CIDR(t *testing.T) {
	ruleHelper := &RuleHelper{}
	ruleHelper.InitRuleEngine()

	// 使用 CIDR 格式
	drls := `
rule R835f9bf09867473dbe873027241db108 "允许CIDR网段访问" salience 10 {
    when
        RF.IPInRanges(MF.SRC_IP, "192.168.1.0/24", "10.0.0.0/8") == true
    then
        Retract("R835f9bf09867473dbe873027241db108");
}`

	var ruleconfigs []model.Rules
	rule := model.Rules{
		HostCode:        "",
		RuleCode:        "R835f9bf09867473dbe873027241db108",
		RuleName:        "允许CIDR网段访问",
		RuleContent:     drls,
		RuleContentJSON: "",
		RuleVersionName: "1.0",
		RuleVersion:     1,
		IsPublicRule:    0,
		IsManualRule:    1,
		RuleStatus:      1,
	}
	ruleconfigs = append(ruleconfigs, rule)

	_, err := ruleHelper.LoadRules(ruleconfigs)
	if err != nil {
		t.Errorf("加载规则失败: %v", err)
		return
	}

	// 测试用例
	testCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"192.168.1网段", "192.168.1.100", true},
		{"10网段", "10.0.0.1", true},
		{"10网段大范围", "10.255.255.255", true},
		{"不在范围内", "192.168.2.1", false},
		{"不在范围内2", "11.0.0.1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			webLog := &innerbean.WebLog{
				SRC_IP: tc.ip,
			}

			ruleMatches, err := ruleHelper.Match("MF", webLog)
			if err != nil {
				t.Errorf("规则匹配错误: %v", err)
				return
			}

			matched := len(ruleMatches) > 0
			if matched != tc.expected {
				t.Errorf("IP %s: 期望匹配=%v, 实际匹配=%v", tc.ip, tc.expected, matched)
			} else {
				t.Logf("✓ IP %s: 匹配结果正确 (匹配=%v)", tc.ip, matched)
			}
		})
	}
}

// TestUserRule_ReversedLogic 测试用户规则 - 反转逻辑（禁止内网访问）
func TestUserRule_ReversedLogic(t *testing.T) {
	ruleHelper := &RuleHelper{}
	ruleHelper.InitRuleEngine()

	// 如果用户想要禁止这些内网IP访问
	drls := `
rule R835f9bf09867473dbe873027241db107 "禁止特定内网网段访问" salience 10 {
    when
        RF.IPInRanges(MF.SRC_IP, "172.16.0.0-172.20.255.254", "192.168.0.0-192.168.1.254") == false
    then
        Retract("R835f9bf09867473dbe873027241db107");
}`

	var ruleconfigs []model.Rules
	rule := model.Rules{
		HostCode:        "",
		RuleCode:        "R835f9bf09867473dbe873027241db107",
		RuleName:        "禁止特定内网网段访问",
		RuleContent:     drls,
		RuleContentJSON: "",
		RuleVersionName: "1.0",
		RuleVersion:     1,
		IsPublicRule:    0,
		IsManualRule:    1,
		RuleStatus:      1,
	}
	ruleconfigs = append(ruleconfigs, rule)

	_, err := ruleHelper.LoadRules(ruleconfigs)
	if err != nil {
		t.Errorf("加载规则失败: %v", err)
		return
	}

	// 内网IP不应该匹配（因为我们是禁止内网）
	webLog1 := &innerbean.WebLog{
		SRC_IP: "172.18.0.1",
	}

	ruleMatches1, _ := ruleHelper.Match("MF", webLog1)
	if len(ruleMatches1) > 0 {
		t.Errorf("内网IP 172.18.0.1 不应该匹配规则（应该被排除）")
	} else {
		t.Logf("✓ 内网IP正确未匹配（被禁止）")
	}

	// 外网IP应该匹配
	webLog2 := &innerbean.WebLog{
		SRC_IP: "8.8.8.8",
	}

	ruleMatches2, _ := ruleHelper.Match("MF", webLog2)
	if len(ruleMatches2) == 0 {
		t.Errorf("外网IP 8.8.8.8 应该匹配规则")
	} else {
		t.Logf("✓ 外网IP正确匹配")
	}
}

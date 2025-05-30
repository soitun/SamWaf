package cache

import (
	"testing"
	"time"
)

func TestWafCache_SetWithTTl(t *testing.T) {
	wafcache := InitWafCache()
	wafcache.SetWithTTl("KEY1", "我是key1的值", 5*time.Second)
	time.Sleep(65 * time.Second)
	key1Value := wafcache.Get("KEY1")
	if str, ok := key1Value.(string); ok {
		println(str)
	}
	time.Sleep(65 * time.Second)

}
func TestWafCache_GetExpireTime(t *testing.T) {
	wafcache := InitWafCache()
	wafcache.SetWithTTl("KEY1", "我是key1的值", 5*time.Minute)
	key1Value, err := wafcache.GetExpireTime("KEY1")
	if err == nil {
		println(key1Value.String())
	}
}

func TestWafCache_GetExpireTimeForever(t *testing.T) {
	wafcache := InitWafCache()
	wafcache.Set("KEY1", "我是key1的值")
	key1Value, err := wafcache.GetExpireTime("KEY1")
	if err == nil {
		println(key1Value.String())
	}
}
func TestWafCache_GetString(t *testing.T) {
	wafcache := InitWafCache()
	wafcache.SetWithTTl("KEY1", "我是key1的值字符串", 5*time.Second)
	key1Value, err := wafcache.GetString("KEY1")
	if err == nil {
		println(key1Value)
	}
}
func TestWafCache_IsKeyExist(t *testing.T) {
	wafcache := InitWafCache()
	bExist := wafcache.IsKeyExist("KEY1")
	if bExist {
		println("存在")
	} else {
		println("不存在")
	}
}

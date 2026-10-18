package server

import (
	"os"
	"testing"
	"time"
)

// 仅供显式浏览器测试：真实 Kratos HTTP + 内存仓储；绝不用于正式服务或宣称 MySQL 已验证。
func TestBrowserHarness(t *testing.T) {
	if os.Getenv("DAILYWINS_BROWSER_TEST") != "1" {
		t.Skip("仅显式开启浏览器联调夹具")
	}
	server, _ := newDailyServer(t)
	if err := os.WriteFile("../../../.cache/browser-api-url.txt", []byte(server.URL), 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("浏览器测试 HTTP 夹具已启动，URL 已写入 .cache/browser-api-url.txt")
	<-time.After(20 * time.Minute)
}

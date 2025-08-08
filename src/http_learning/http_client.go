package http_learning

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const (
	URL_ADDRESS_REDIRECT = "https://httpbin.org/redirect/2"
	URL_ADDRESS_COOKIES  = "https://httpbin.org/cookies/set/sessionid/abc123"
)

func ClientWithRedirect() {

	jar, _ := cookiejar.New(nil)

	customTransport := &http.Transport{
		MaxIdleConns:       10,               // 最大空闲连接数
		IdleConnTimeout:    30 * time.Second, // 空闲连接超时时间
		DisableCompression: true,             // 禁用压缩
	}
	// 自定义重定向
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("重定向次数过多")
			}
			fmt.Printf("重定向到: %s\n", req.URL.String())
			return nil
		},
		Transport: customTransport,
		Jar:       jar,
		Timeout:   30 * time.Second,
	}

	resp, err := client.Get(URL_ADDRESS_COOKIES)

	if err != nil {
		fmt.Println("请求失败:", err)
		return
	}
	defer resp.Body.Close()

	// 检查cookies
	u, _ := url.Parse("https://httpbin.org")
	cookies := jar.Cookies(u)

	fmt.Println("状态码:", resp.StatusCode)
	fmt.Println("响应头:", resp.Header)
	fmt.Printf("保存的Cookies数量: %d\n", len(cookies))

	for i, cookie := range cookies {
		fmt.Printf("Cookie %d: %s=%s\n", i+1, cookie.Name, cookie.Value)
	}

	// 第二次请求：验证cookie是否被发送
	fmt.Println("\n--- 第二次请求，验证cookie ---")
	resp2, err := client.Get("https://httpbin.org/cookies")
	if err != nil {
		fmt.Println("第二次请求失败:", err)
		return
	}
	defer resp2.Body.Close()

	// 读取响应体查看cookies
	body := make([]byte, 1024)
	n, _ := resp2.Body.Read(body)
	fmt.Printf("服务器收到的cookies: %s\n", string(body[:n]))
}

func DemonstrateClientDo() {
	// 创建自定义客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 1. GET 请求
	req1, _ := http.NewRequest("GET", "https://httpbin.org/get", nil)
	req1.Header.Set("User-Agent", "My-Custom-Client/1.0")
	resp1, err := client.Do(req1)
	if err != nil {
		fmt.Printf("GET 请求失败: %v\n", err)
		return
	}
	defer resp1.Body.Close()
	fmt.Printf("GET 响应状态: %s\n", resp1.Status)

	// 2. POST 请求
	body := strings.NewReader(`{"name":"张三","age":25}`)
	req2, _ := http.NewRequest("POST", "https://httpbin.org/post", body)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer token123")
	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Printf("POST 请求失败: %v\n", err)
		return
	}
	defer resp2.Body.Close()
	fmt.Printf("POST 响应状态: %s\n", resp2.Status)

	// 3. PUT 请求
	updateBody := strings.NewReader(`{"status":"updated"}`)
	req3, _ := http.NewRequest("PUT", "https://httpbin.org/put", updateBody)
	req3.Header.Set("Content-Type", "application/json")
	resp3, err := client.Do(req3)
	if err != nil {
		fmt.Printf("PUT 请求失败: %v\n", err)
		return
	}
	defer resp3.Body.Close()
	fmt.Printf("PUT 响应状态: %s\n", resp3.Status)

	// 4. DELETE 请求
	req4, _ := http.NewRequest("DELETE", "https://httpbin.org/delete", nil)
	resp4, err := client.Do(req4)
	if err != nil {
		fmt.Printf("DELETE 请求失败: %v\n", err)
		return
	}
	defer resp4.Body.Close()
	fmt.Printf("DELETE 响应状态: %s\n", resp4.Status)
}

func DemonstrateProxy() {
	// 代理服务器地址 (请替换为你的实际代理地址)
	proxyURL, err := url.Parse("http://your_proxy_address:your_proxy_port")
	if err != nil {
		fmt.Printf("解析代理URL失败: %v\n", err)
		return
	}

	// 创建自定义 Transport 并设置 Proxy 字段
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL), // 使用固定的代理地址
		// 或者使用环境变量代理：Proxy: http.ProxyFromEnvironment,
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}

	// 创建使用自定义 Transport 的 Client
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// 发送请求，该请求将通过配置的代理发送
	resp, err := client.Get("https://httpbin.org/ip") // 访问一个会显示请求IP的网站
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体失败: %v\n", err)
		return
	}

	fmt.Printf("响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应体:\n%s\n", string(body))

	// 如果响应体中的IP是你的代理服务器IP，说明代理设置成功。
}

func DemonstrateCompression() {
	// 1. 启用压缩（默认行为）
	fmt.Println("=== 启用压缩 ===")
	transportWithCompression := &http.Transport{
		DisableCompression: false, // 默认值
	}

	clientWithCompression := &http.Client{
		Transport: transportWithCompression,
	}

	// 添加请求头检查
	req1, _ := http.NewRequest("GET", "https://httpbin.org/gzip", nil)
	fmt.Printf("请求头 Accept-Encoding: %s\n", req1.Header.Get("Accept-Encoding"))

	resp1, err := clientWithCompression.Do(req1)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp1.Body.Close()

	fmt.Printf("Content-Encoding: %s\n", resp1.Header.Get("Content-Encoding"))
	fmt.Printf("Content-Length: %s\n", resp1.Header.Get("Content-Length"))

	body1, _ := io.ReadAll(resp1.Body)
	fmt.Printf("最终响应体长度: %d 字节\n", len(body1))
	fmt.Printf("响应体前100字符: %s\n\n", string(body1)[:min(100, len(body1))])

	// 2. 禁用压缩
	fmt.Println("=== 禁用压缩 ===")
	transportWithoutCompression := &http.Transport{
		DisableCompression: true, // 禁用自动压缩处理
	}

	clientWithoutCompression := &http.Client{
		Transport: transportWithoutCompression,
	}

	req2, _ := http.NewRequest("GET", "https://httpbin.org/gzip", nil)
	fmt.Printf("请求头 Accept-Encoding: %s\n", req2.Header.Get("Accept-Encoding"))

	resp2, err := clientWithoutCompression.Do(req2)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	fmt.Printf("Content-Encoding: %s\n", resp2.Header.Get("Content-Encoding"))
	fmt.Printf("Content-Length: %s\n", resp2.Header.Get("Content-Length"))

	body2, _ := io.ReadAll(resp2.Body)
	fmt.Printf("原始响应体长度: %d 字节\n", len(body2))

	// 手动解压缩
	if resp2.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(bytes.NewReader(body2)) // 使用 bytes.NewReader
		if err != nil {
			fmt.Printf("创建gzip读取器失败: %v\n", err)
			return
		}
		defer gzipReader.Close()

		uncompressed, err := io.ReadAll(gzipReader)
		if err != nil {
			fmt.Printf("解压缩失败: %v\n", err)
			return
		}

		fmt.Printf("手动解压缩后长度: %d 字节\n", len(uncompressed))
		fmt.Printf("解压缩后前100字符: %s\n", string(uncompressed)[:min(100, len(uncompressed))])

		// 对比两次解压缩后的内容
		if len(body1) == len(uncompressed) {
			fmt.Println("✅ 两次请求解压缩后长度一致")
		} else {
			fmt.Printf("⚠️  两次请求内容不同: %d vs %d 字节\n", len(body1), len(uncompressed))
		}
	}
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package config

import (
    "bufio"
    "fmt"
    "net/url"
    "os"
    "strconv"
    "strings"
)

type ProxyConfig struct {
    ServerHost string
    ServerPort int
    ProxyURL   string
    ProxyHost  string
    ProxyPort  int
    ProxyUser  string
    ProxyPass  string
}

type Config struct {
    Proxies []ProxyConfig
}

func LoadConfig() *Config {
    // Đọc danh sách proxies từ file
    proxies, err := loadProxiesFromFile("list_proxy.txt")
    if err != nil {
        panic("Failed to load proxies: " + err.Error())
    }
    
    return &Config{
        Proxies: proxies,
    }
}

func loadProxiesFromFile(filename string) ([]ProxyConfig, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var proxies []ProxyConfig
    scanner := bufio.NewScanner(file)
    port := 3000 // Bắt đầu từ port 3000
    
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        parts := strings.Split(line, ":")
        if len(parts) != 4 {
            continue // Bỏ qua dòng không đúng format
        }
        
        proxyHost := parts[0]
        proxyPort, err := strconv.Atoi(parts[1])
        if err != nil {
            continue // Bỏ qua nếu port không hợp lệ
        }
        
        proxyUser := parts[2]
        proxyPass := parts[3]
        
        proxyURL := fmt.Sprintf("http://%s:%s@%s:%d", 
            url.QueryEscape(proxyUser), 
            url.QueryEscape(proxyPass), 
            proxyHost, 
            proxyPort)
        
        proxyConfig := ProxyConfig{
            ServerHost: "0.0.0.0",
            ServerPort: port,
            ProxyURL:   proxyURL,
            ProxyHost:  proxyHost,
            ProxyPort:  proxyPort,
            ProxyUser:  proxyUser,
            ProxyPass:  proxyPass,
        }

        fmt.Println("Cau hinh ", port, proxyConfig)
        
        proxies = append(proxies, proxyConfig)
        port++ // Tăng port cho proxy tiếp theo
    }
    
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    
    return proxies, nil
}

func (c *ProxyConfig) GetServerAddress() string {
    return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}

func (c *ProxyConfig) GetProxyAddress() string {
    return fmt.Sprintf("%s:%d", c.ProxyHost, c.ProxyPort)
}
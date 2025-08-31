package config

import (
    "fmt"
    "net/url"
    "os"
    "strconv"
    "strings"
)

type Config struct {
    ServerHost string
    ServerPort int
    ProxyURL   string
    ProxyHost  string
    ProxyPort  int
    ProxyUser  string
    ProxyPass  string
}

func LoadConfig() *Config {
    // Địa chỉ server của bạn
    serverHost := getEnv("SERVER_HOST", "192.168.1.75")
    serverPort := getEnvAsInt("SERVER_PORT", 3000)
    
    // Thông tin proxy thật
    proxyInfo := getEnv("PROXY_INFO", "160.30.191.243:27711:conghung:conghung")
    
    // Parse proxy info
    parts := strings.Split(proxyInfo, ":")
    if len(parts) != 4 {
        panic("Invalid proxy format. Expected: IP:PORT:USERNAME:PASSWORD")
    }
    
    proxyHost := parts[0]
    proxyPort, err := strconv.Atoi(parts[1])
    if err != nil {
        panic("Invalid proxy port")
    }
    
    proxyUser := parts[2]
    proxyPass := parts[3]
    
    // Tạo proxy URL với authentication
    proxyURL := fmt.Sprintf("http://%s:%s@%s:%d", 
        url.QueryEscape(proxyUser), 
        url.QueryEscape(proxyPass), 
        proxyHost, 
        proxyPort)
    
    return &Config{
        ServerHost: serverHost,
        ServerPort: serverPort,
        ProxyURL:   proxyURL,
        ProxyHost:  proxyHost,
        ProxyPort:  proxyPort,
        ProxyUser:  proxyUser,
        ProxyPass:  proxyPass,
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func (c *Config) GetServerAddress() string {
    return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}

func (c *Config) GetProxyAddress() string {
    return fmt.Sprintf("%s:%d", c.ProxyHost, c.ProxyPort)
}
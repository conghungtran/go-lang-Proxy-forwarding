package main

import (
    "crypto/tls"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
)

func main() {
    // Test URLs - thử cả HTTP và HTTPS
    testURLs := []string{
        "http://httpbin.org/ip", // HTTP trước
        "https://httpbin.org/ip", // HTTPS sau
        "http://facebook.com", // HTTP Facebook
        "https://facebook.com", // HTTPS Facebook
    }
    
    proxyURL, _ := url.Parse("http://192.168.1.75:3000")
    
    client := &http.Client{
        Transport: &http.Transport{
            Proxy: http.ProxyURL(proxyURL),
            TLSClientConfig: &tls.Config{
                InsecureSkipVerify: false,
            },
        },
        // Xử lý redirects
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            fmt.Printf("Redirect: %s -> %s\n", via[len(via)-1].URL, req.URL)
            if len(via) >= 5 {
                return fmt.Errorf("too many redirects")
            }
            return nil
        },
    }
    
    for _, testURL := range testURLs {
        fmt.Printf("\nTesting: %s\n", testURL)
        
        resp, err := client.Get(testURL)
        if err != nil {
            if strings.Contains(err.Error(), "redirect") {
                fmt.Printf("Redirect error: %v\n", err)
            } else {
                fmt.Printf("Error: %v\n", err)
            }
            continue
        }
        
        fmt.Printf("Final URL: %s\n", resp.Request.URL)
        fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
        fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
        fmt.Printf("Location: %s\n", resp.Header.Get("Location"))
        
        // Chỉ đọc body nếu không phải redirect
        if resp.StatusCode < 300 || resp.StatusCode >= 400 {
            body, err := io.ReadAll(resp.Body)
            resp.Body.Close()
            
            if err != nil {
                fmt.Printf("Error reading body: %v\n", err)
                continue
            }
            
            fmt.Printf("Body length: %d\n", len(body))
            if len(body) < 500 {
                fmt.Printf("Body: %s\n", string(body))
            }
        }
        
        fmt.Println("----------------------------------------")
    }
}
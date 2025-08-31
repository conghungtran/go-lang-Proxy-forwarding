package main

import (
    "crypto/tls"
    "fmt"
    "io"
    "net/http"
    "net/url"
)

func main() {
    // Test HTTPS resources
    testURLs := []string{
        "https://static.xx.fbcdn.net/rsrc.php/v3/yO/l/0,cross/6AXRrkTNwMR.css",
        "https://static.xx.fbcdn.net/rsrc.php/v3/yD/l/0,cross/IJ89ZRILpqa.css",
        "https://static.xx.fbcdn.net/rsrc.php/v4/yJ/r/EwSS5svXugt.js",
        "https://httpbin.org/ip",
    }
    
    proxyURL, _ := url.Parse("http://192.168.1.75:3000")
    
    client := &http.Client{
        Transport: &http.Transport{
            Proxy: http.ProxyURL(proxyURL),
            TLSClientConfig: &tls.Config{
                InsecureSkipVerify: false,
            },
        },
    }
    
    for _, testURL := range testURLs {
        fmt.Printf("Testing: %s\n", testURL)
        
        resp, err := client.Get(testURL)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        body, err := io.ReadAll(resp.Body)
        resp.Body.Close()
        
        if err != nil {
            fmt.Printf("Error reading body: %v\n", err)
            continue
        }
        
        fmt.Printf("Status: %d, Content-Type: %s, Length: %d\n", 
            resp.StatusCode, resp.Header.Get("Content-Type"), len(body))
        fmt.Println("----------------------------------------")
    }
}
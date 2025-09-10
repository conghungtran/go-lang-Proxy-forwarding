package main

import (
    "crypto/tls"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "sync"
)

func main() {
    testURLs := []string{
        "http://httpbin.org/ip",
        "https://httpbin.org/ip",
        "http://facebook.com",
        "https://www.facebook.com",
    }
    
    var wg sync.WaitGroup
    results := make(map[int]string)
    
    // Test từng proxy từ port 3000 đến 3009
    for port := 3000; port <= 3009; port++ {
        wg.Add(1)
        
        go func(proxyPort int) {
            defer wg.Done()
            
            proxyURL, _ := url.Parse(fmt.Sprintf("http://192.168.1.75:%d", proxyPort))
            
            client := &http.Client{
                Transport: &http.Transport{
                    Proxy: http.ProxyURL(proxyURL),
                    TLSClientConfig: &tls.Config{
                        InsecureSkipVerify: false,
                    },
                },
            }
            
            var success bool
            for _, testURL := range testURLs {
                resp, err := client.Get(testURL)
                if err != nil {
                    continue
                }
                
                body, _ := io.ReadAll(resp.Body)
                resp.Body.Close()
                
                if resp.StatusCode == 200 && len(body) > 0 {
                    success = true
                    break
                }
            }
            
            if success {
                results[proxyPort] = "✅ WORKING"
            } else {
                results[proxyPort] = "❌ FAILED"
            }
        }(port)
    }
    
    wg.Wait()
    
    fmt.Println("\n=== PROXY TEST RESULTS ===")
    for port := 3000; port <= 3009; port++ {
        fmt.Printf("Port %d: %s\n", port, results[port])
    }
}
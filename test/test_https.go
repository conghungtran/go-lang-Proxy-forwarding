package main

import (
    "fmt"
    "io"
    "net/http"
)

func main() {
    fmt.Println("Testing direct connection to target sites...")
    
    testURLs := []string{
        "http://httpbin.org/ip",
        "http://httpbin.org/html",
        "http://httpbin.org/json",
    }
    
    client := &http.Client{}
    
    for _, url := range testURLs {
        fmt.Printf("\nTesting: %s\n", url)
        
        resp, err := client.Get(url)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
        fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
        
        body, err := io.ReadAll(resp.Body)
        resp.Body.Close()
        
        if err != nil {
            fmt.Printf("Read error: %v\n", err)
            continue
        }
        
        fmt.Printf("Length: %d bytes\n", len(body))
        if len(body) < 200 {
            fmt.Printf("Preview: %s\n", string(body))
        }
    }
}
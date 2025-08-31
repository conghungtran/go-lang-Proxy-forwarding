package handler

import (
    
    "crypto/tls"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "proxy-server/config"
    "proxy-server/utils"
    "strings"
    "time"

    "go.uber.org/zap"
)

type ProxyHandler struct {
    config *config.Config
    client *http.Client
}

func NewProxyHandler(cfg *config.Config) *ProxyHandler {
    proxyURL, err := url.Parse(cfg.ProxyURL)
    if err != nil {
        utils.GetLogger().Fatal("Failed to parse proxy URL", zap.Error(err))
    }
    
    // Tạo HTTP client với proxy authentication
    transport := &http.Transport{
        Proxy: http.ProxyURL(proxyURL),
        DialContext: (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        MaxIdleConns:          100,
        IdleConnTimeout:       90 * time.Second,
        TLSHandshakeTimeout:   15 * time.Second,
        ExpectContinueTimeout: 2 * time.Second,
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: false,
            MinVersion:         tls.VersionTLS12,
        },
    }
    
    client := &http.Client{
        Transport: transport,
        Timeout:   120 * time.Second,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            // Cho phép redirects nhưng giới hạn
            if len(via) >= 10 {
                return fmt.Errorf("too many redirects")
            }
            return nil
        },
    }
    
    return &ProxyHandler{
        config: cfg,
        client: client,
    }
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // logger := utils.GetLogger().With(
    //     zap.String("method", r.Method),
    //     zap.String("url", r.URL.String()),
    //     zap.String("remote_addr", r.RemoteAddr),
    //     zap.String("host", r.Host),
    // )
    
    if r.Method == "CONNECT" {
        h.handleHTTPS(w, r)
        return
    }
    
    h.handleHTTP(w, r)
}

func (h *ProxyHandler) handleHTTP(w http.ResponseWriter, r *http.Request) {
    logger := utils.GetLogger().With(
        zap.String("method", r.Method),
        zap.String("url", r.URL.String()),
    )
    
    logger.Info("Processing HTTP request")
    
    // Xây dựng target URL
    targetURL := h.buildTargetURL(r)
    if targetURL == "" {
        http.Error(w, "Cannot determine target URL", http.StatusBadRequest)
        return
    }
    
    // Tạo request mới
    proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
    if err != nil {
        logger.Error("Failed to create proxy request", zap.Error(err))
        http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
        return
    }
    
    // Copy headers
    h.copyRequestHeaders(r, proxyReq)
    
    // Thêm headers quan trọng
    h.addRequiredHeaders(proxyReq)
    
    start := time.Now()
    resp, err := h.client.Do(proxyReq)
    if err != nil {
        logger.Error("Failed to send request through proxy", 
            zap.Error(err),
            zap.String("target", targetURL),
        )
        http.Error(w, "Failed to connect through proxy: "+err.Error(), http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()
    
    duration := time.Since(start)
    
    logger.Info("Received response", 
        zap.Int("status", resp.StatusCode),
        zap.Duration("duration", duration),
        zap.String("content_type", resp.Header.Get("Content-Type")),
    )
    
    // Copy response headers
    h.copyResponseHeaders(w, resp)
    
    // Set status code
    w.WriteHeader(resp.StatusCode)
    
    // Copy response body
    written, err := io.Copy(w, resp.Body)
    if err != nil {
        logger.Error("Failed to copy response body", zap.Error(err))
    } else {
        logger.Info("Response sent successfully", 
            zap.Int64("bytes", written),
        )
    }
}

func (h *ProxyHandler) handleHTTPS(w http.ResponseWriter, r *http.Request) {
    logger := utils.GetLogger().With(
        zap.String("method", r.Method),
        zap.String("url", r.URL.String()),
    )
    
    logger.Info("Processing HTTPS CONNECT request")
    
    // Kết nối trực tiếp đến destination
    destConn, err := net.DialTimeout("tcp", r.URL.Host, 30*time.Second)
    if err != nil {
        logger.Error("Failed to connect to destination", zap.Error(err))
        http.Error(w, "Failed to connect to destination", http.StatusBadGateway)
        return
    }
    defer destConn.Close()
    
    // Trả về 200 OK
    w.WriteHeader(http.StatusOK)
    
    // Hijack connection
    hijacker, ok := w.(http.Hijacker)
    if !ok {
        logger.Error("Hijacking not supported")
        http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
        return
    }
    
    clientConn, _, err := hijacker.Hijack()
    if err != nil {
        logger.Error("Failed to hijack connection", zap.Error(err))
        http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
        return
    }
    defer clientConn.Close()
    
    logger.Info("HTTPS tunnel established")
    
    // Thiết lập tunnel
    go h.copyData(destConn, clientConn)
    h.copyData(clientConn, destConn)
}

func (h *ProxyHandler) copyData(dst, src net.Conn) {
    defer dst.Close()
    defer src.Close()
    
    io.Copy(dst, src)
}

func (h *ProxyHandler) buildTargetURL(r *http.Request) string {
    scheme := "http"
    if r.TLS != nil {
        scheme = "https"
    }
    
    host := r.Host
    if host == "" {
        host = r.URL.Host
    }
    if host == "" {
        return ""
    }
    
    // Xây dựng URL đầy đủ
    if strings.Contains(r.URL.String(), "://") {
        return r.URL.String()
    }
    
    return scheme + "://" + host + r.URL.String()
}

func (h *ProxyHandler) addRequiredHeaders(req *http.Request) {
    // Set default headers nếu chưa có
    headers := map[string]string{
        "User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        "Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.5",
        "Accept-Encoding": "gzip, deflate, br",
        "Connection":      "keep-alive",
    }
    
    for key, value := range headers {
        if req.Header.Get(key) == "" {
            req.Header.Set(key, value)
        }
    }
}

func (h *ProxyHandler) copyRequestHeaders(src, dst *http.Request) {
    for name, values := range src.Header {
        // Skip problematic headers
        if strings.EqualFold(name, "Proxy-Connection") || 
           strings.EqualFold(name, "Connection") ||
           strings.EqualFold(name, "Keep-Alive") {
            continue
        }
        
        for _, value := range values {
            dst.Header.Add(name, value)
        }
    }
    
    // Set Host header
    if src.Host != "" {
        dst.Header.Set("Host", src.Host)
        dst.Host = src.Host
    }
}

func (h *ProxyHandler) copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
    for name, values := range resp.Header {
        // Skip problematic headers
        if strings.EqualFold(name, "Connection") ||
           strings.EqualFold(name, "Keep-Alive") ||
           strings.EqualFold(name, "Proxy-Authenticate") ||
           strings.EqualFold(name, "Proxy-Authorization") {
            continue
        }
        
        for _, value := range values {
            w.Header().Add(name, value)
        }
    }
}
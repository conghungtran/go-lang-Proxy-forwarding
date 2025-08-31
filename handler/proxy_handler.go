package handler

import (
    "crypto/tls"
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
        Timeout:   60 * time.Second,
    }
    
    return &ProxyHandler{
        config: cfg,
        client: client,
    }
}

func (h *ProxyHandler) HandleHTTP(w http.ResponseWriter, r *http.Request) {
    logger := utils.GetLogger().With(
        zap.String("method", r.Method),
        zap.String("url", r.URL.String()),
        zap.String("remote_addr", r.RemoteAddr),
        zap.String("host", r.Host),
    )
    
    // Xử lý HTTPS CONNECT requests
    if r.Method == "CONNECT" {
        h.HandleConnect(w, r)
        return
    }
    
    logger.Info("Received HTTP request")
    
    targetURL := h.constructTargetURL(r)
    if targetURL == "" {
        http.Error(w, "Cannot determine target URL", http.StatusBadRequest)
        return
    }
    
    proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
    if err != nil {
        logger.Error("Failed to create proxy request", zap.Error(err))
        http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
        return
    }
    defer r.Body.Close()
    
    h.copyHeaders(r, proxyReq)
    h.addImportantHeaders(proxyReq)
    
    start := time.Now()
    resp, err := h.client.Do(proxyReq)
    if err != nil {
        logger.Error("Failed to send request to proxy", 
            zap.Error(err),
            zap.String("target_url", targetURL),
        )
        http.Error(w, "Failed to connect to proxy: "+err.Error(), http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()
    
    duration := time.Since(start)
    logger.Info("Proxy response received", 
        zap.Int("status_code", resp.StatusCode),
        zap.Duration("duration", duration),
        zap.String("content_type", resp.Header.Get("Content-Type")),
    )
    
    h.copyResponseHeaders(w, resp)
    w.WriteHeader(resp.StatusCode)
    
    written, err := io.Copy(w, resp.Body)
    if err != nil {
        logger.Error("Failed to copy response body", zap.Error(err))
    } else {
        logger.Info("Response sent to client", 
            zap.Int64("bytes_written", written),
        )
    }
}

// Sửa lại hàm HandleConnect để xử lý đúng HTTPS
func (h *ProxyHandler) HandleConnect(w http.ResponseWriter, r *http.Request) {
    logger := utils.GetLogger().With(
        zap.String("method", r.Method),
        zap.String("url", r.URL.String()),
        zap.String("remote_addr", r.RemoteAddr),
    )
    
    logger.Info("Received HTTPS CONNECT request", zap.String("target", r.URL.Host))
    
    // Kết nối trực tiếp đến destination thay vì qua proxy
    destConn, err := net.DialTimeout("tcp", r.URL.Host, 30*time.Second)
    if err != nil {
        logger.Error("Failed to connect to destination", zap.Error(err))
        http.Error(w, "Failed to connect to destination", http.StatusBadGateway)
        return
    }
    defer destConn.Close()
    
    // Trả về response 200 cho client
    w.WriteHeader(http.StatusOK)
    
    // Hijack connection để thiết lập tunnel
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
    
    // Đã gửi response 200 ở trên, không cần gửi lại
    logger.Info("HTTPS tunnel established", zap.String("target", r.URL.Host))
    
    // Thiết lập tunnel giữa client và destination
    go h.copyIO(destConn, clientConn, "client->dest")
    h.copyIO(clientConn, destConn, "dest->client")
}

func (h *ProxyHandler) copyIO(dst, src net.Conn, direction string) {
    defer dst.Close()
    defer src.Close()
    
    written, err := io.Copy(dst, src)
    if err != nil {
        utils.GetLogger().Debug("Copy IO completed", 
            zap.String("direction", direction),
            zap.Int64("bytes_written", written),
            zap.Error(err),
        )
    }
}

func (h *ProxyHandler) addImportantHeaders(req *http.Request) {
    if req.Header.Get("User-Agent") == "" {
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
    }
    
    if req.Header.Get("Accept") == "" {
        req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
    }
    
    if req.Header.Get("Accept-Language") == "" {
        req.Header.Set("Accept-Language", "en-US,en;q=0.9")
    }
    
    if req.Header.Get("Accept-Encoding") == "" {
        req.Header.Set("Accept-Encoding", "gzip, deflate, br")
    }
    
    if req.Header.Get("Connection") == "" {
        req.Header.Set("Connection", "keep-alive")
    }
}

func (h *ProxyHandler) copyHeaders(src *http.Request, dst *http.Request) {
    for name, values := range src.Header {
        if name == "Proxy-Connection" || name == "Connection" {
            continue
        }
        for _, value := range values {
            dst.Header.Add(name, value)
        }
    }
    
    if src.Host != "" {
        dst.Header.Set("Host", src.Host)
        dst.Host = src.Host
    }
}

func (h *ProxyHandler) copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
    for name, values := range resp.Header {
        if name == "Connection" {
            continue
        }
        
        // Fix Cross-Origin issues
        if name == "Cross-Origin-Opener-Policy" || name == "Cross-Origin-Embedder-Policy" {
            continue // Bỏ qua các headers gây lỗi
        }
        
        for _, value := range values {
            w.Header().Add(name, value)
        }
    }
    
    // Thêm headers để fix mixed content
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (h *ProxyHandler) constructTargetURL(r *http.Request) string {
    scheme := "http"
    if r.TLS != nil {
        scheme = "https"
    }
    
    host := r.Host
    if host == "" && r.URL.Host != "" {
        host = r.URL.Host
    }
    
    if host == "" {
        return ""
    }
    
    targetURL := r.URL.String()
    if !strings.Contains(targetURL, "://") {
        targetURL = scheme + "://" + host + targetURL
    }
    
    return targetURL
}
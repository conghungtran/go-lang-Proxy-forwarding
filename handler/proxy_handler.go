package handler

import (
	"bufio"
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

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
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

	// Debug: log all incoming headers
	for name, values := range r.Header {
		logger.Debug("Incoming header", zap.String("name", name), zap.Strings("values", values))
	}

	// ... rest of the function

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

	// Copy headers
	h.copyHeaders(r, proxyReq)

	// Add important headers
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

	// Copy response headers FIRST
	h.copyResponseHeaders(w, resp)

	// THEN set status code
	w.WriteHeader(resp.StatusCode)

	// FINALLY copy body
	written, err := io.Copy(w, resp.Body)
	if err != nil {
		logger.Error("Failed to copy response body", zap.Error(err))
	} else {
		logger.Info("Response sent to client",
			zap.Int64("bytes_written", written),
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
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	}

	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "gzip, deflate")
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
	}
}

func (h *ProxyHandler) copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for name, values := range resp.Header {
		if name == "Connection" {
			continue
		}
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
}

func (h *ProxyHandler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger().With(
		zap.String("method", r.Method),
		zap.String("url", r.URL.String()),
		zap.String("remote_addr", r.RemoteAddr),
	)

	logger.Info("Received HTTPS CONNECT request")

	proxyConn, err := net.DialTimeout("tcp", h.config.GetProxyAddress(), 30*time.Second)
	if err != nil {
		logger.Error("Failed to connect to proxy", zap.Error(err))
		http.Error(w, "Failed to connect to proxy", http.StatusBadGateway)
		return
	}
	defer proxyConn.Close()

	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n",
		r.URL.Host, r.URL.Host)
	_, err = proxyConn.Write([]byte(connectReq))
	if err != nil {
		logger.Error("Failed to send CONNECT to proxy", zap.Error(err))
		return
	}

	reader := bufio.NewReader(proxyConn)
	resp, err := http.ReadResponse(reader, r)
	if err != nil || resp.StatusCode != 200 {
		logger.Error("Proxy refused CONNECT", zap.Int("status", resp.StatusCode))
		http.Error(w, "Proxy refused CONNECT", http.StatusBadGateway)
		return
	}

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

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	logger.Info("HTTPS tunnel established", zap.String("target", r.URL.Host))

	go io.Copy(proxyConn, clientConn)
	io.Copy(clientConn, proxyConn)
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
		// For proxy check tools, use a default
		host = "httpbin.org"
	}

	targetURL := r.URL.String()
	if !strings.Contains(targetURL, "://") {
		targetURL = scheme + "://" + host + targetURL
	}

	return targetURL
}

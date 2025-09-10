package auth

import (
	"encoding/base64"
	"net/http"
	"proxy-server/config"
	"proxy-server/utils"
	"strings"

	"go.uber.org/zap"
)

// ProxyAuthenticator xử lý authentication cho proxy
type ProxyAuthenticator struct {
	config *config.ProxyConfig
}

// NewProxyAuthenticator tạo authenticator mới
func NewProxyAuthenticator(cfg *config.ProxyConfig) *ProxyAuthenticator {
	return &ProxyAuthenticator{
		config: cfg,
	}
}

// Authenticate kiểm tra xác thực từ client
func (a *ProxyAuthenticator) Authenticate(r *http.Request) bool {
	logger := utils.GetLogger()
	
	// Nếu không require auth, cho phép tất cả
	if !a.config.RequireAuth {
		return true
	}

	// Lấy Proxy-Authorization header
	authHeader := r.Header.Get("Proxy-Authorization")
	if authHeader == "" {
		logger.Debug("No Proxy-Authorization header found")
		return false
	}

	// Parse Basic authentication
	if !strings.HasPrefix(authHeader, "Basic ") {
		logger.Debug("Invalid auth header format", zap.String("header", authHeader))
		return false
	}

	// Decode base64 credentials
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		logger.Debug("Failed to decode auth header", zap.Error(err))
		return false
	}

	// Parse username:password
	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		logger.Debug("Invalid credentials format", zap.String("credentials", credentials))
		return false
	}

	username, password := parts[0], parts[1]

	// Kiểm tra credentials
	if username == a.config.AuthUser && password == a.config.AuthPass {
        logger.Info("Authentication successful", 
            zap.String("user", username),
            zap.Int("proxy_port", a.config.ServerPort))
		return true
	}

    logger.Warn("Authentication failed", 
        zap.String("provided_user", username),
        zap.String("expected_user", a.config.AuthUser),
        zap.Int("proxy_port", a.config.ServerPort))
	return false
}

// RequireAuth trả về 407 Proxy Authentication Required
func (a *ProxyAuthenticator) RequireAuth(w http.ResponseWriter) {
	logger := utils.GetLogger()
	
	logger.Info("Sending 407 Proxy Authentication Required")
	
	// Set headers cho proxy authentication
	w.Header().Set("Proxy-Authenticate", "Basic realm=\"Proxy Server\"")
	w.Header().Set("Content-Type", "text/plain")
	
	// Trả về 407 status
	w.WriteHeader(http.StatusProxyAuthRequired)
	w.Write([]byte("407 Proxy Authentication Required\r\n\r\nThis proxy server requires authentication.\r\nPlease provide valid credentials using Proxy-Authorization header."))
}

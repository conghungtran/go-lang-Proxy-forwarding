package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "proxy-server/config"
    "proxy-server/handler"
    "proxy-server/utils"
    "syscall"
    "time"

    "go.uber.org/zap"
)

func main() {
    if err := utils.InitLogger(); err != nil {
        panic("Failed to initialize logger: " + err.Error())
    }
    defer utils.GetLogger().Sync()
    
    logger := utils.GetLogger()
    
    cfg := config.LoadConfig()
    
    logger.Info("Starting proxy server",
        zap.String("address", cfg.GetServerAddress()),
        zap.String("proxy", cfg.GetProxyAddress()),
    )
    
    // Tạo proxy handler
    proxyHandler := handler.NewProxyHandler(cfg)
    
    // Tạo server
    server := &http.Server{
        Addr:         cfg.GetServerAddress(),
        Handler:      proxyHandler, // Sử dụng handler trực tiếp
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }
    
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        logger.Info("Server listening", zap.String("address", server.Addr))
        
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Server failed", zap.Error(err))
        }
    }()
    
    <-stop
    logger.Info("Shutting down server...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        logger.Error("Shutdown failed", zap.Error(err))
    }
    
    logger.Info("Server stopped")
}
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "proxy-server/config"
    "proxy-server/handler"
    "proxy-server/utils"
    "sync"
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
    
    if len(cfg.Proxies) == 0 {
        logger.Fatal("No proxies configured")
    }
    
    logger.Info("Starting multiple proxy servers", 
        zap.Int("proxy_count", len(cfg.Proxies)))
    
    var wg sync.WaitGroup
    servers := make([]*http.Server, len(cfg.Proxies))
    
    // Khởi động server cho mỗi proxy
    for i, proxyCfg := range cfg.Proxies {
        wg.Add(1)
        
        go func(index int, cfg config.ProxyConfig) {
            defer wg.Done()
            
            logger := utils.GetLogger().With(
                zap.Int("proxy_index", index),
                zap.String("server_address", cfg.GetServerAddress()),
                zap.String("proxy_address", cfg.GetProxyAddress()),
            )
            
            // Tạo proxy handler cho proxy này
            proxyHandler := handler.NewProxyHandler(&cfg)
            
            server := &http.Server{
                Addr:         cfg.GetServerAddress(),
                Handler:      proxyHandler,
                ReadTimeout:  30 * time.Second,
                WriteTimeout: 30 * time.Second,
                IdleTimeout:  120 * time.Second,
            }
            
            servers[index] = server
            
            logger.Info("Starting proxy server")
            
            if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                logger.Error("Server failed", zap.Error(err))
            }
        }(i, proxyCfg)
        
        // Chờ một chút giữa các server để tránh conflict
        time.Sleep(100 * time.Millisecond)
    }
    
    // Wait for interrupt signal
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop
    
    logger.Info("Shutting down all servers...")
    
    // Graceful shutdown cho tất cả servers
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    for i, server := range servers {
        if server != nil {
            logger.Info("Shutting down server", 
                zap.Int("proxy_index", i),
                zap.String("address", server.Addr))
            
            if err := server.Shutdown(ctx); err != nil {
                logger.Error("Server shutdown failed", 
                    zap.Int("proxy_index", i),
                    zap.Error(err))
            }
        }
    }
    
    wg.Wait()
    logger.Info("All servers stopped")
}
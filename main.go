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

    "github.com/gorilla/mux"
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
        zap.String("server", cfg.GetServerAddress()),
        zap.String("proxy", cfg.GetProxyAddress()),
    )
    
    proxyHandler := handler.NewProxyHandler(cfg)
    
    router := mux.NewRouter()
    
    // Route cho tất cả các request
    router.PathPrefix("/").HandlerFunc(proxyHandler.HandleHTTP)
    
    server := &http.Server{
        Addr:         cfg.GetServerAddress(),
        Handler:      router,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }
    
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Failed to start server", zap.Error(err))
        }
    }()
    
    <-stop
    logger.Info("Shutting down server...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        logger.Error("Server shutdown failed", zap.Error(err))
    }
    
    logger.Info("Server stopped")
}
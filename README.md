# Proxy Forward Server

Multi-proxy forwarding server written in Go that supports HTTP/HTTPS proxy authentication.

## Features

✅ **Multiple Proxy Servers**: Run multiple proxy servers simultaneously on different ports  
✅ **Proxy Authentication**: Username/password authentication required for each proxy  
✅ **HTTP/HTTPS Support**: Full support for both HTTP and HTTPS CONNECT tunneling  
✅ **Detailed Logging**: Comprehensive logging with Zap logger  
✅ **Graceful Shutdown**: Clean shutdown with Ctrl+C  
✅ **High Performance**: Concurrent connections and efficient connection handling  

## Quick Start

### 1. Build the Project

```bash
# Using Makefile (recommended)
make build

# Or manually
go build -o bin/proxy-forward main.go
```

### 2. Configure Proxies

Edit `list_proxy.txt` with your upstream proxy list:
```
host:port:username:password
103.179.189.225:24069:Proxy:Proxy
103.179.189.235:10449:user10449:9296178958
```

### 3. Start the Server

```bash
# Using the start script
./start-proxy.sh

# Or directly
./bin/proxy-forward

# Or using Makefile
make run
```

## Authentication Credentials

Each proxy server requires authentication. Check `proxy_credentials.txt` for the credentials:

| Port | Username | Password | Upstream Proxy |
|------|----------|----------|----------------|
| 3000 | user3000 | pass3000 | 103.179.189.225:24069 |
| 3001 | user3001 | pass3001 | 103.179.189.235:10449 |
| 3002 | user3002 | pass3002 | 103.179.189.235:11533 |
| ... | ... | ... | ... |

## Usage Examples

### 1. Using curl
```bash
curl -x http://user3000:pass3000@localhost:3000 http://httpbin.org/ip
```

### 2. Using wget
```bash
wget --proxy-user=user3000 --proxy-password=pass3000 \
     --proxy=http://localhost:3000 http://httpbin.org/ip
```

### 3. Browser Configuration
- **Proxy Server**: localhost:3000
- **Username**: user3000
- **Password**: pass3000

### 4. Python requests
```python
import requests

proxies = {
    'http': 'http://user3000:pass3000@localhost:3000',
    'https': 'http://user3000:pass3000@localhost:3000'
}

response = requests.get('http://httpbin.org/ip', proxies=proxies)
print(response.json())
```

## Build Commands

```bash
# Build for current platform
make build

# Build for Linux
make build-linux

# Build for Windows
make build-windows

# Build for macOS
make build-mac

# Build for all platforms
make build-all

# Clean build files
make clean

# Install dependencies
make deps
```

## Project Structure

```
├── bin/                    # Compiled binaries
├── auth/                   # Authentication module
├── config/                 # Configuration module
├── handler/               # HTTP/HTTPS handlers
├── utils/                 # Utility functions
├── main.go                # Main application
├── list_proxy.txt         # Upstream proxy list
├── proxy_credentials.txt  # Authentication credentials
├── start-proxy.sh        # Start script
└── Makefile              # Build commands
```

## Configuration

### Upstream Proxy Format
File: `list_proxy.txt`
```
host:port:username:password
103.179.189.225:24069:Proxy:Proxy
103.179.189.235:10449:user10449:9296178958
```

### Generated Authentication
- Username: `user{port}` (e.g., user3000)
- Password: `pass{port}` (e.g., pass3000)
- Each proxy server gets a unique port starting from 3000

## Logs

The server provides detailed JSON logs including:
- Authentication success/failure
- HTTP/HTTPS request processing
- Connection establishment
- Error handling

Example log:
```json
{
  "level": "info",
  "timestamp": "2025-09-10T11:14:14.485+0700",
  "msg": "Authentication successful",
  "user": "user3000",
  "proxy_port": 3000
}
```

## Security

- ✅ All proxy connections require authentication
- ✅ Credentials are generated per port
- ✅ Failed authentication attempts are logged
- ✅ 407 Proxy Authentication Required returned for invalid credentials

## Performance

- Supports thousands of concurrent connections
- Efficient connection pooling
- Minimal memory footprint (~6.6MB binary)
- Fast startup time

## Troubleshooting

### Common Issues

1. **Port already in use**
   - Change starting port in config or stop conflicting services

2. **Authentication failed**
   - Check credentials in `proxy_credentials.txt`
   - Ensure client supports proxy authentication

3. **Connection timeout**
   - Verify upstream proxy is accessible
   - Check firewall settings

### Debug Mode
Set log level to debug for detailed troubleshooting:
```go
// In utils/logger.go
config := zap.NewDevelopmentConfig()
```

## License

This project is licensed under the MIT License.

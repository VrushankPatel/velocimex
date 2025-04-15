# Developer Guide

This guide provides comprehensive information for developers who want to contribute to or extend the Velocimex platform.

## Table of Contents
- [Development Environment](#development-environment)
- [Building from Source](#building-from-source)
- [Testing](#testing)
- [Contributing Guidelines](#contributing-guidelines)
- [Plugin Development](#plugin-development)
- [Performance Optimization](#performance-optimization)

## Development Environment

### Prerequisites
- Go 1.21 or later
- Node.js 18 or later
- Docker and Docker Compose
- Git
- Make

### Setting Up the Environment

1. **Clone the Repository**
```bash
git clone https://github.com/velocimex/velocimex.git
cd velocimex
```

2. **Install Dependencies**
```bash
# Install Go dependencies
go mod download

# Install Node.js dependencies
cd frontend
npm install
```

3. **Environment Variables**
Create a `.env` file in the root directory:
```bash
VELOCIMEX_ENV=development
VELOCIMEX_DEBUG=true
VELOCIMEX_DB_HOST=localhost
VELOCIMEX_DB_PORT=5432
VELOCIMEX_DB_USER=postgres
VELOCIMEX_DB_PASSWORD=your_password
```

## Building from Source

### Backend
```bash
# Build the backend
make build-backend

# Run tests
make test-backend

# Run linter
make lint-backend
```

### Frontend
```bash
# Build the frontend
make build-frontend

# Run tests
make test-frontend

# Run linter
make lint-frontend
```

### Docker
```bash
# Build all containers
make docker-build

# Run the application
make docker-up
```

## Testing

### Unit Tests
```bash
# Run all unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/marketdata
```

### Integration Tests
```bash
# Run integration tests
make test-integration

# Run specific integration test
go test -tags=integration ./tests/integration/marketdata
```

### Performance Tests
```bash
# Run performance benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkOrderBook ./pkg/marketdata
```

## Contributing Guidelines

### Code Style
- Follow Go standard formatting (`go fmt`)
- Use meaningful variable and function names
- Write comprehensive comments
- Follow the project's coding conventions

### Git Workflow
1. **Create a Branch**
```bash
git checkout -b feature/your-feature-name
```

2. **Make Changes**
- Write code
- Add tests
- Update documentation

3. **Commit Changes**
```bash
git add .
git commit -m "feat: add new feature"
```

4. **Push Changes**
```bash
git push origin feature/your-feature-name
```

5. **Create Pull Request**
- Fill out the PR template
- Request review
- Address feedback

### Documentation
- Update relevant documentation
- Add examples where necessary
- Include API changes
- Document new features

## Plugin Development

### Creating a New Plugin
1. **Create Plugin Structure**
```bash
mkdir -p plugins/your-plugin
cd plugins/your-plugin
```

2. **Implement Plugin Interface**
```go
package yourplugin

import (
    "github.com/velocimex/plugin"
)

type YourPlugin struct {
    // Plugin fields
}

func (p *YourPlugin) Initialize(config plugin.Config) error {
    // Initialize plugin
    return nil
}

func (p *YourPlugin) Start() error {
    // Start plugin
    return nil
}

func (p *YourPlugin) Stop() error {
    // Stop plugin
    return nil
}
```

3. **Register Plugin**
```go
func init() {
    plugin.Register("your-plugin", func() plugin.Plugin {
        return &YourPlugin{}
    })
}
```

### Plugin Configuration
```yaml
plugins:
  your-plugin:
    enabled: true
    config:
      setting1: value1
      setting2: value2
```

## Performance Optimization

### Backend Optimization
1. **Profiling**
```bash
# Generate CPU profile
go test -cpuprofile=cpu.prof -bench=.

# Generate memory profile
go test -memprofile=mem.prof -bench=.
```

2. **Benchmarking**
```go
func BenchmarkOrderBook(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Benchmark code
    }
}
```

### Frontend Optimization
1. **Bundle Analysis**
```bash
npm run build -- --analyze
```

2. **Performance Monitoring**
```javascript
// Add performance monitoring
performance.mark('start');
// Your code
performance.mark('end');
performance.measure('operation', 'start', 'end');
```

### Database Optimization
1. **Indexing**
```sql
CREATE INDEX idx_trades_timestamp ON trades(timestamp);
CREATE INDEX idx_orders_status ON orders(status);
```

2. **Query Optimization**
```sql
EXPLAIN ANALYZE SELECT * FROM trades WHERE timestamp > NOW() - INTERVAL '1 day';
```

## Best Practices

1. **Code Quality**
   - Write clean, maintainable code
   - Follow SOLID principles
   - Use design patterns appropriately

2. **Testing**
   - Write comprehensive tests
   - Maintain high test coverage
   - Include edge cases

3. **Documentation**
   - Keep documentation up-to-date
   - Include examples
   - Document API changes

4. **Security**
   - Follow security best practices
   - Regular security audits
   - Input validation

5. **Performance**
   - Profile regularly
   - Optimize critical paths
   - Monitor resource usage

For more information about specific components, refer to the [Technical Documentation](../technical/index.md). 
.PHONY: build run run-bg stop status logs test clean install init fmt vet help

# 初始化配置
init:
	@if [ ! -f configs/config.yaml ]; then \
		cp configs/config.example.yaml configs/config.yaml; \
		echo "Created configs/config.yaml from example"; \
	else \
		echo "configs/config.yaml already exists"; \
	fi

# 构建
build:
	go build -o haystack-lite main.go

# 运行
run: init
	@echo "Starting haystack-lite..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@sleep 1
	go run main.go

# 运行（后台模式）
run-bg: init build
	@echo "Starting haystack-lite in background..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@sleep 1
	@./haystack-lite > /tmp/haystack.log 2>&1 &
	@echo "Started with PID: $$!"
	@echo "Logs: tail -f /tmp/haystack.log"

# 停止后台进程
stop:
	@echo "Stopping haystack-lite..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@pkill -9 haystack-lite 2>/dev/null || true
	@echo "Stopped"

# 查看日志
logs:
	@tail -f /tmp/haystack.log

# 查看状态
status:
	@if lsof -ti:8080 > /dev/null 2>&1; then \
		echo "✓ haystack-lite is running on port 8080"; \
		echo "PID: $$(lsof -ti:8080)"; \
	else \
		echo "✗ haystack-lite is not running"; \
	fi

# 安装依赖
install:
	go mod download

# 测试
test:
	chmod +x scripts/test.sh
	./scripts/test.sh

# 清理
clean:
	rm -f haystack-lite
	rm -rf data/
	rm -f test_file.txt downloaded_file.txt

# 格式化代码
fmt:
	go fmt ./...

# 检查代码
vet:
	go vet ./...

# 帮助
help:
	@echo "可用命令:"
	@echo "  make init       - 初始化配置文件"
	@echo "  make install    - 安装依赖"
	@echo "  make build      - 编译程序"
	@echo "  make run        - 运行程序（前台，自动关闭旧进程）"
	@echo "  make run-bg     - 运行程序（后台）"
	@echo "  make stop       - 停止后台进程"
	@echo "  make status     - 查看运行状态"
	@echo "  make logs       - 查看日志"
	@echo "  make test       - 运行测试"
	@echo "  make clean      - 清理文件"
	@echo "  make fmt        - 格式化代码"
	@echo "  make vet        - 检查代码"

# 保留完整 Go 工具链，供容器内构建、测试和调试使用。
FROM golang:1.22

WORKDIR /app

# 本项目没有第三方模块；仍预下载模块以保证构建步骤可离线重复执行。
COPY go.mod ./
RUN go mod download

COPY . .
RUN go build ./...

CMD ["bash"]

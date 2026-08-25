# 单阶段构建：与仓库既有项目（task151/task147）对齐的工具链版本。
FROM golang:1.26.3-bookworm

ENV CGO_ENABLED=0 \
    GOTOOLCHAIN=local \
    GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOTOOLCHAIN=local go build -o /out/sensealign ./cmd/sensealign

WORKDIR /data
ENTRYPOINT ["/out/sensealign"]
CMD ["--smoke-test"]

# 评测构建用 Dockerfile（与 Dockerfile 同源，供 build_benzhi_docker.sh 引用）。
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

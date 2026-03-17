FROM golang:1.25.7 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /plugin .

FROM alpine:3.20
COPY --from=builder /plugin /plugin
ENTRYPOINT ["cp", "/plugin", "/plugins/vnode-plugin"]

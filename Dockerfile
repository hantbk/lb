FROM golang:1.16.3-alpine3.13 as builder
WORKDIR /app
COPY main.go go.mod ./
RUN CGO_ENABLED=0 GOOS=linux go build -o lb .

FROM alpine:3.13
RUN apk add --no-cache ca-certificates
WORKDIR /root
COPY --from=builder /app/lb ./
ENTRYPOINT [ "/root/lb" ]
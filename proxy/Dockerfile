FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o proxy main.go

FROM alpine:3.19

WORKDIR /root/
COPY --from=builder /app/proxy .

EXPOSE 80

CMD ["./proxy"]
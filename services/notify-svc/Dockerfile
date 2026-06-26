FROM golang:1.22-alpine AS builder
WORKDIR /workspace

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/notify-svc/go.mod services/notify-svc/go.sum ./services/notify-svc/
RUN cd services/notify-svc && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /notify-svc ./services/notify-svc/cmd/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /notify-svc /usr/local/bin/notify-svc
EXPOSE 8083 9083
CMD ["notify-svc"]

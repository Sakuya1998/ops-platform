FROM golang:1.22-alpine AS builder
WORKDIR /workspace

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/auth-svc/go.mod services/auth-svc/go.sum ./services/auth-svc/
RUN cd services/auth-svc && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-svc ./services/auth-svc/cmd/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /auth-svc /usr/local/bin/auth-svc
EXPOSE 8080 9080
CMD ["auth-svc"]

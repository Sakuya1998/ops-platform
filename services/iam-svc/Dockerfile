FROM golang:1.22-alpine AS builder
WORKDIR /workspace

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/iam-svc/go.mod services/iam-svc/go.sum ./services/iam-svc/
RUN cd services/iam-svc && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /iam-svc ./services/iam-svc/cmd/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /iam-svc /usr/local/bin/iam-svc
EXPOSE 8081 9081
CMD ["iam-svc"]

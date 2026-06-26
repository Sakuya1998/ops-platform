FROM golang:1.22-alpine AS builder
WORKDIR /workspace

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/audit-svc/go.mod services/audit-svc/go.sum ./services/audit-svc/
RUN cd services/audit-svc && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /audit-svc ./services/audit-svc/cmd/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /audit-svc /usr/local/bin/audit-svc
EXPOSE 8082 9082
CMD ["audit-svc"]

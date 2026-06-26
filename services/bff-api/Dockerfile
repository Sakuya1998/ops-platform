FROM golang:1.22-alpine AS builder
WORKDIR /workspace

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/bff-api/go.mod services/bff-api/go.sum ./services/bff-api/
RUN cd services/bff-api && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bff-api ./services/bff-api/cmd/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /bff-api /usr/local/bin/bff-api
EXPOSE 8070
CMD ["bff-api"]

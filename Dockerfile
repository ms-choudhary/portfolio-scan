#########
FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download 

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -tags timetzdata -o ./portfolio-scan .

#########

FROM alpine:3.21 

# Install CA certificates so KiteConnect works
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/portfolio-scan /usr/local/bin/portfolio-scan
COPY ui/dist /app/ui/dist

EXPOSE 9876
CMD ["portfolio-scan"]

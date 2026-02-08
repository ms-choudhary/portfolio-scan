FROM node:22-alpine AS ui-builder
WORKDIR /app

COPY ui/package*.json ./ui/
RUN cd ui && npm install

COPY ui/ ./ui/
RUN cd ui && npm run build


FROM golang:1.24.0 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download 

COPY . .

COPY --from=ui-builder /app/ui/dist ./ui/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -tags timetzdata -o ./portfolio-scan .

FROM alpine:3.21 

# Install CA certificates so KiteConnect works
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/portfolio-scan /usr/local/bin/portfolio-scan

EXPOSE 9876
CMD ["portfolio-scan"]

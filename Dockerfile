#########
FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download 

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o ./portfolio-scan .

#########
FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /app/portfolio-scan /usr/local/bin/portfolio-scan
COPY ui/dist /app/ui/dist

EXPOSE 9876

CMD ["portfolio-scan"]

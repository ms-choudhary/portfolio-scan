FROM alpine:3.14

WORKDIR /app

COPY portfolio-scan /app/portfolio-scan
COPY ui/dist /app/ui/dist

CMD ["/app/portfolio-scan"]

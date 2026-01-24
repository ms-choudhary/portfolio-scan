FROM alpine:3.14

WORKDIR /app

RUN apk add --no-cache ca-certificates openssl wget && \
    wget http://crt.sectigo.com/SectigoPublicServerAuthenticationCADVR36.crt && \
    openssl x509 -inform DER -in SectigoPublicServerAuthenticationCADVR36.crt -out /usr/local/share/ca-certificates/sectigo.crt && \
    rm SectigoPublicServerAuthenticationCADVR36.crt && \
    update-ca-certificates

COPY portfolio-scan /app/portfolio-scan
COPY ui/dist /app/ui/dist

CMD ["/app/portfolio-scan"]

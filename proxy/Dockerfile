FROM ubuntu:22.04

ARG DEBIAN_FRONTEND=noninteractive

EXPOSE 3128

RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y squid-openssl openssl && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc/squid/certs && \
    openssl genrsa -out /etc/squid/certs/server.key 2048 && \
    openssl req -x509 -new -days 3650 \
        -subj "/C=US/CN=squid-proxy" \
        -addext "subjectAltName=DNS:squid-proxy,DNS:localhost,IP:127.0.0.1" \
        -key /etc/squid/certs/server.key \
        -out /etc/squid/certs/server.crt && \
    chmod 600 /etc/squid/certs/server.key

ARG CADDY_VERSION=2.7.6

FROM golang:latest AS builder
ARG SUPERVISORD_VERSION=master

#
# Build our go apps ( gossipy + supervisord)
#
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN GOOS=linux GOARCH=arm64 CGO_ENABLED=0 make gossipy/build

RUN git clone --depth 1 https://github.com/NixM0nk3y/supervisord.git -b ${SUPERVISORD_VERSION} /tmp/supervisord \
    && cd /tmp/supervisord \
    && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /app/supervisord

FROM --platform=linux/arm64 caddy:${CADDY_VERSION}

#
RUN echo "==> Installing extra packages..." \
    && apk add --no-cache bash curl jq

# drop in our extra config and scripts
COPY Caddyfile /etc/caddy/Caddyfile
COPY scripts/boot.sh /app/boot.sh
COPY supervisor/supervisord.conf /etc/supervisord.conf

# copy in our go apps
COPY --from=builder /app/gossipy /app/gossipy
COPY --from=builder /app/supervisord /app/supervisord

EXPOSE 8080/tcp
EXPOSE 7947/tcp

# start out supervisord process
CMD [ "/app/supervisord","-c","/etc/supervisord.conf" ]

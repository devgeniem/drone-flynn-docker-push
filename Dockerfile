FROM docker:1.11-dind

ARG FLYNN_L=/usr/local/bin/flynn

# Install flynn client
RUN apk add --no-cache curl && \
    curl -sSL -A "`uname -sp`" https://dl.flynn.io/cli | zcat >$FLYNN_L && \
    chmod +x $FLYNN_L && \
    apk del curl

ADD drone-flynn-docker /bin/

ENTRYPOINT ["/usr/local/bin/dockerd-entrypoint.sh", "/bin/drone-flynn-docker"]

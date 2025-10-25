FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY bin/aura-linux ./aura
COPY configs ./configs

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1

ENTRYPOINT ["./aura"]
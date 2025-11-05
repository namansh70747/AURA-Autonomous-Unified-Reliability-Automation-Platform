FROM alpine:latest
# yahan maine basic linux image copy ki hai

RUN apk --no-cache add ca-certificates curl
# no-cache se container chhota banta hai
# ca-certificates se app HTTPS APIs ko safely call kar sakta hai
# curl se app localhost pe healthcheck kar sakta hai


WORKDIR /app

COPY bin/aura-linux ./aura
COPY configs ./configs

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1

ENTRYPOINT ["./aura"]
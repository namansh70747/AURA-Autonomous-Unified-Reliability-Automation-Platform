FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY bin/sample-app-linux ./sample-app

EXPOSE 8080

CMD ["./sample-app"]


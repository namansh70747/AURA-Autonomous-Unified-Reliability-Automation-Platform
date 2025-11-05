FROM alpine:latest

RUN apk --no-cache add ca-certificates wget curl

# what it does:
# runs a command inside container to install some tools
# why helpful:
# ca-certificates → helps your app call HTTPS APIs safely
# wget → helps if your app needs to download files
# --no-cache → makes container smaller

#├── bin
#├── etc
#├── home
#├── lib
#├── root    <--- this is where your WORKDIR /root is
#├── tmp
#├── usr
#└── var
WORKDIR /root/
# 1) / (single slash)
# / means start from main root of container.
COPY bin/sample-app-linux ./sample-app
# ./means current folder mai sample app ka file bna usme sample-app-linux pa do 

EXPOSE 8080
# Expose the port 8080 to the outside world

CMD ["./sample-app"]


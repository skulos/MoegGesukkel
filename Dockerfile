FROM golang:latest

WORKDIR /root

COPY . .

RUN go build -o fileServer server/server.go

EXPOSE 8080

CMD [ "/bin/bash" ]
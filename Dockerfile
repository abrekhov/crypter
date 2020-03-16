FROM golang:alpine
RUN apk update && apk add git
WORKDIR /go/src/github.com/abrekhov/crypter
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

ENV PORT=8000
ENV ADDRESS=0.0.0.0

CMD [ "crypter" ]
FROM golang:latest

WORKDIR /go/src/app
COPY . .

RUN go get github.com/neo4j/neo4j-go-driver/v4/neo4j
RUN go get github.com/pilu/fresh

CMD [ "fresh" ]

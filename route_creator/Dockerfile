FROM golang:latest

WORKDIR /go/src/app

# Gerekli dosyalar container'a kopyalanır.
COPY . .

# Container için gerekli golang paketleri kurulur. 
RUN go get github.com/neo4j/neo4j-go-driver/v4/neo4j
RUN go get github.com/pilu/fresh

# Golang fresh paketi ile go modülümüz çalıştırılır ve değişiklikleri algılanır rebuild edilebilir.
CMD [ "fresh" ]

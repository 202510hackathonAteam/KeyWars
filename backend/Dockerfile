FROM golang:1.25.2-alpine3.22

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/air-verse/air@latest

COPY . .

COPY wait-for.sh /wait-for.sh
RUN chmod +x /wait-for.sh

ENV GO111MODULE=on
CMD ["/wait-for.sh", "db", "3306", "air", "-c", ".air.toml"]
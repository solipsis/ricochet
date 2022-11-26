FROM golang:1.17-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o ricochet-robotbot

CMD [ "./ricochet-robotbot" ]


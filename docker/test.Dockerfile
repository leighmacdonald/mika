FROM golang:1.14-alpine
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
RUN apk add make build-base git
# Set the Current Working Directory inside the container
WORKDIR /mika

COPY go.mod go.sum ./
RUN go mod download
COPY . .

CMD make test

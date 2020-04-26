FROM golang:latest
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"

# Set the Current Working Directory inside the container
WORKDIR /mika

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download
COPY . .
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && go build -ldflags "-X consts.VersionHash=$GIT_COMMIT"

EXPOSE 34001
EXPOSE 34000

# Command to run the executable
CMD ["./mika", "serve"]

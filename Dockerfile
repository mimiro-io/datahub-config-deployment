FROM golang:1.16.0

# mim cli version
ARG CLI_VERSION=0.4.2

# Install mim cli
RUN curl -L https://github.com/mimiro-io/datahub-cli/releases/download/${CLI_VERSION}/datahub-cli_${CLI_VERSION}_Linux_x86_64.tar.gz -o cli.tar.gz
RUN mkdir -p /cli
RUN tar -xzf cli.tar.gz -C /cli
ENV PATH="/cli:${PATH}"

# Set the Current Working Directory inside the container
WORKDIR /deploy

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go vet ./...
RUN go build -o bin/mim-deploy ./cmd/deploy/main.go
ENV PATH="/deploy/bin:${PATH}"

ENTRYPOINT ["mim-deploy"]

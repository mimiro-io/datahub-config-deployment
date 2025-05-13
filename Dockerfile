FROM golang:1.24-alpine3.21

# mim cli version
ARG CLI_VERSION=0.20.1

# datahub-tslib version
ARG TSLIB_VERSION=0.2.0

# Install mim cli
RUN wget https://github.com/mimiro-io/datahub-cli/releases/download/v${CLI_VERSION}/datahub-cli_${CLI_VERSION}_Linux_x86_64.tar.gz -O cli.tar.gz
RUN mkdir -p /cli
RUN tar -xzf cli.tar.gz -C /cli
ENV PATH="/cli:${PATH}"

# Set the Current Working Directory inside the container
WORKDIR /deploy

# install node for typescript support
RUN set -uex; \
    apk update; \
    apk add ca-certificates curl gnupg git; \
    mkdir -p /etc/apt/keyrings; \
    mkdir -p /etc/apt/sources.list.d/; \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key \
     | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg; \
    NODE_MAJOR=18; \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" \
     > /etc/apt/sources.list.d/nodesource.list; \
    apk update; \
    apk add nodejs npm; \
    node --version; \
    npm init -y; \
    ls -la; \
    npm config set update-notifier false; \
    npm install mimiro-io/datahub-tslib#${TSLIB_VERSION} --save-dev

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

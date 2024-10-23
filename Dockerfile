# Use an official Golang image as the base image
FROM golang:1.21

# Install make and other necessary build tools
RUN apt-get update && apt-get install -y make build-essential

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files first to leverage Docker cache
COPY go.mod go.sum ./

# Download Go module dependencies
RUN go mod download

# Copy the entire project to the working directory
COPY . .

# Accept the MODE argument for selecting the environment
ARG MODE=dev

# Run make with the specified mode
RUN make MODE=${MODE}

# Expose the necessary ports
EXPOSE 80
EXPOSE 443
EXPOSE 2210

# Command to run the Go app with the config file
CMD ["/app/build/shiroxy", "-c", "/app/defaults/shiroxy.conf.yaml"]

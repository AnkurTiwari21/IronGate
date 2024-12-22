# Use official Golang image as the base image
FROM golang:1.23

# Set the working directory in the container
WORKDIR /app

# Copy the go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod tidy

# Copy the entire project folder (including subdirectories and all Go files)
COPY . .

# Build the Go application (building the entire project)
RUN go build -o main .

# Define the command to run the executable
CMD [ "./main" ]

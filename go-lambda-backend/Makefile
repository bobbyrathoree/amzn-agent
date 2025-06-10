.PHONY: build clean test chatf botsf knowledgef websocketf

# Go build flags
LDFLAGS=-ldflags "-s -w"
GOFLAGS=-trimpath

# Build directories
LAMBDA_DIR=../lambda/functions

# Function names
FUNCTIONS=chat bots knowledge websocket

# Default target
build: $(FUNCTIONS)

# Build all functions
chat: chatf
bots: botsf
knowledge: knowledgef
websocket: websocketf

# Build specific functions
chatf:
	mkdir -p $(LAMBDA_DIR)/chat
	cd cmd/functions/chat && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o ../../../$(LAMBDA_DIR)/chat/bootstrap ./main.go
	chmod +x $(LAMBDA_DIR)/chat/bootstrap

botsf:
	mkdir -p $(LAMBDA_DIR)/bots
	cd cmd/functions/bots && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o ../../../$(LAMBDA_DIR)/bots/bootstrap ./main.go
	chmod +x $(LAMBDA_DIR)/bots/bootstrap

knowledgef:
	mkdir -p $(LAMBDA_DIR)/knowledge
	cd cmd/functions/knowledge && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o ../../../$(LAMBDA_DIR)/knowledge/bootstrap ./main.go
	chmod +x $(LAMBDA_DIR)/knowledge/bootstrap

websocketf:
	mkdir -p $(LAMBDA_DIR)/websocket
	cd cmd/functions/websocket && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o ../../../$(LAMBDA_DIR)/websocket/bootstrap ./main.go
	chmod +x $(LAMBDA_DIR)/websocket/bootstrap

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f $(LAMBDA_DIR)/chat/bootstrap
	rm -f $(LAMBDA_DIR)/bots/bootstrap
	rm -f $(LAMBDA_DIR)/knowledge/bootstrap
	rm -f $(LAMBDA_DIR)/websocket/bootstrap
build:
	@go build -o bin/fs

run: build
	@./bin/fs

# Run in interactive mode
run-interactive: build
	@./bin/fs -interactive

# Run demo mode
run-demo: build
	@./bin/fs -demo

# Example: Run node on port 3000
run-node1: build
	@./bin/fs -addr :3000 -interactive

# Example: Run node on port 4000 and connect to node1
run-node2: build
	@./bin/fs -addr :4000 -bootstrap localhost:3000 -interactive

# Example: Run node on port 5000 and connect to both previous nodes
run-node3: build
	@./bin/fs -addr :5000 -bootstrap localhost:3000,localhost:4000 -interactive

# Clean up storage directories
clean-storage:
	@echo "Cleaning storage directories..."
	@rm -rf storage/ *_network/ *.data/ *.storage/ 2>/dev/null || true
	@echo "Storage cleaned"

test:
	@go test ./...
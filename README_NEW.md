# PeerVault

PeerVault is a **P2P distributed file storage system** that allows you to store and retrieve files across a distributed network of nodes. It uses encryption to secure your files and ensures redundancy by broadcasting files to multiple peers in the network and makes sure they're available even if one computer goes offline.

---

## What It Does

- **Distributed File Storage**: Files are stored across multiple nodes in the network.
- **Encryption**: Files are encrypted before being stored and decrypted when retrieved.
- **Streaming Support**: Handles large files efficiently using streaming.
- **Custom Protocol**: Uses a lightweight custom protocol for communication between nodes.
- **Peer Discovery**: Automatically connects to other nodes in the network for redundancy.
- **Network Configuration**: Dynamic peer connections across different machines on the same network.

---

## Flow
![flow](https://github.com/user-attachments/assets/b409604d-eddf-4b39-8f9d-d098e96e0f07)

## Quick Start

### 1. Install
1. Download the project:
   ```bash
   git clone https://github.com/AdityaKrSingh26/PeerVault.git
   cd PeerVault
   ```

2. Build the project:
   ```bash
   make build
   ```

### 2. Run on Single Machine (Testing)

Start multiple nodes on different ports:
```bash
# Terminal 1: First node
./bin/fs -addr :3000 -interactive

# Terminal 2: Second node (connects to first)
./bin/fs -addr :4000 -bootstrap localhost:3000 -interactive

# Terminal 3: Third node (connects to both)
./bin/fs -addr :5000 -bootstrap localhost:3000,localhost:4000 -interactive
```

### 3. Run Across Multiple Machines (Real Network)

**On Computer 1 (e.g., IP: 192.168.1.100):**
```bash
./bin/fs -addr :3000 -interactive
```

**On Computer 2 (e.g., IP: 192.168.1.101):**
```bash
./bin/fs -addr :3000 -bootstrap 192.168.1.100:3000 -interactive
```

**On Computer 3 (e.g., IP: 192.168.1.102):**
```bash
./bin/fs -addr :3000 -bootstrap 192.168.1.100:3000,192.168.1.101:3000 -interactive
```

### 4. Interactive Commands

Once in interactive mode, use these commands:
```
PeerVault> store myfile.txt      # Store a file
PeerVault> get myfile.txt        # Retrieve a file
PeerVault> status               # Show connection status
PeerVault> quit                 # Exit
```

---

## Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:3000` | Listen address (e.g., `:3000`, `:4000`) |
| `-bootstrap` | `""` | Comma-separated bootstrap nodes |
| `-interactive` | `false` | Run in interactive mode |
| `-demo` | `false` | Run demo with test data |
| `-h` | | Show help |

### Examples

**Start a node:**
```bash
./bin/fs -addr :3000 -interactive
```

**Connect to existing network:**
```bash
./bin/fs -addr :4000 -bootstrap 192.168.1.100:3000 -interactive
```

**Run demo mode:**
```bash
./bin/fs -addr :3000 -demo
```

---

## Network Setup

For detailed instructions on setting up PeerVault across multiple laptops, see [NETWORK_SETUP_GUIDE.md](NETWORK_SETUP_GUIDE.md).

For a comprehensive technical guide, see [COMPREHENSIVE_GUIDE.md](COMPREHENSIVE_GUIDE.md).

---

## Example Multi-Machine Setup

### Step 1: Find IP Addresses
On each machine, find the local IP:
```bash
# Windows
ipconfig

# Linux/macOS  
ip addr show
```

### Step 2: Start First Node
On Machine 1 (192.168.1.100):
```bash
./bin/fs -addr :3000 -interactive
```

### Step 3: Connect Other Nodes
On Machine 2 (192.168.1.101):
```bash
./bin/fs -addr :3000 -bootstrap 192.168.1.100:3000 -interactive
```

### Step 4: Test File Sharing
On Machine 1:
```
PeerVault> store document.txt
File 'document.txt' stored successfully
```

On Machine 2:
```
PeerVault> get document.txt
File content: Sample data for file: document.txt
```

---

## Testing

Run tests to make sure everything works:
```bash
make test
```

Run demo mode for quick testing:
```bash
make run-demo
```

---

## Makefile Commands

- `make build` - Build the executable
- `make run` - Run with default settings
- `make run-demo` - Run in demo mode
- `make run-interactive` - Run in interactive mode
- `make test` - Run all tests

---

## Features

✅ **Working Features:**
- P2P file storage and retrieval
- AES-256 encryption
- Network discovery via bootstrap nodes
- Interactive command-line interface
- Cross-platform support (Windows, Linux, macOS)
- Multi-machine networking

🚧 **Planned Features:**
- Web UI for file management
- File versioning
- Enhanced peer discovery
- File compression
- Access control lists

---

## Troubleshooting

**Connection Issues:**
- Check firewall settings
- Verify IP addresses and ports
- Ensure all machines are on same network

**Port Conflicts:**
- Use different ports: `-addr :4000` instead of `:3000`
- Check for other applications using the port

**File Not Found:**
- Ensure the file was stored on a connected peer
- Check network connectivity with `status` command

For more troubleshooting tips, see [NETWORK_SETUP_GUIDE.md](NETWORK_SETUP_GUIDE.md).

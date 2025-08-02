# PeerVault Network Setup Guide

## Running PeerVault Across Multiple Laptops

This guide explains how to run PeerVault nodes on different laptops connected to the same WiFi network.

## Prerequisites

1. All laptops must be connected to the same WiFi network
2. Firewalls should allow TCP connections on the chosen ports
3. Each laptop should have PeerVault built and ready

## Step-by-Step Setup

### Step 1: Find Your IP Addresses

On each laptop, find the local IP address:

**Windows (PowerShell):**
```powershell
ipconfig | findstr "IPv4"
```

**Windows (Command Prompt):**
```cmd
ipconfig
```

**Linux/macOS:**
```bash
ip addr show | grep inet
# or
ifconfig | grep inet
```

Example output: `192.168.1.100`, `192.168.1.101`, etc.

### Step 2: Start the First Node (Bootstrap Node)

On **Laptop 1** (e.g., IP: 192.168.1.100):
```bash
# Build the project
make build

# Start first node on port 3000
./bin/fs -addr :3000 -interactive
```

The node will display:
```
Starting PeerVault server on :3000
Local IP: 192.168.1.100
PeerVault Interactive Mode
Commands:
  store <filename>  - Store a file
  get <filename>    - Retrieve a file
  status            - Show server status
  quit              - Exit

PeerVault> 
```

### Step 3: Start Additional Nodes

On **Laptop 2** (e.g., IP: 192.168.1.101):
```bash
# Connect to Laptop 1
./bin/fs -addr :4000 -bootstrap 192.168.1.100:3000 -interactive
```

On **Laptop 3** (e.g., IP: 192.168.1.102):
```bash
# Connect to both previous nodes
./bin/fs -addr :5000 -bootstrap 192.168.1.100:3000,192.168.1.101:4000 -interactive
```

### Step 4: Test the Network

1. **On Laptop 1**, store a file:
```
PeerVault> store myfile.txt
File 'myfile.txt' stored successfully
```

2. **On Laptop 2**, retrieve the file:
```
PeerVault> get myfile.txt
File content: Sample data for file: myfile.txt (stored at 14:30:25)
```

3. **Check connection status**:
```
PeerVault> status
Server listening on: :4000
Local IP: 192.168.1.101
Connected peers: 1
  - 192.168.1.100:3000
```

## Command Line Options

### Basic Usage
```bash
./bin/fs [options]
```

### Available Options

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:3000` | Listen address (e.g., `:3000`, `:4000`) |
| `-bootstrap` | `""` | Comma-separated list of bootstrap nodes |
| `-interactive` | `false` | Run in interactive mode for file operations |
| `-demo` | `false` | Run demo mode with test data |

### Example Commands

**Start a standalone node:**
```bash
./bin/fs -addr :3000 -interactive
```

**Connect to existing network:**
```bash
./bin/fs -addr :4000 -bootstrap 192.168.1.100:3000 -interactive
```

**Connect to multiple bootstrap nodes:**
```bash
./bin/fs -addr :5000 -bootstrap 192.168.1.100:3000,192.168.1.101:4000 -interactive
```

**Run demo mode:**
```bash
./bin/fs -addr :3000 -demo
```

## Interactive Commands

When running with `-interactive` flag, you can use these commands:

| Command | Description | Example |
|---------|-------------|---------|
| `store <filename>` | Store a file with sample data | `store document.txt` |
| `get <filename>` | Retrieve and display file content | `get document.txt` |
| `status` | Show server information and connected peers | `status` |
| `quit` | Shutdown the server | `quit` |

## Network Topologies

### Simple Network (2 nodes)
```
Laptop 1 (:3000) ◄──► Laptop 2 (:4000)
   192.168.1.100        192.168.1.101
```

**Setup:**
```bash
# Laptop 1
./bin/fs -addr :3000 -interactive

# Laptop 2  
./bin/fs -addr :4000 -bootstrap 192.168.1.100:3000 -interactive
```

### Star Network (3+ nodes)
```
        Laptop 2 (:4000)
           192.168.1.101
                │
Laptop 1 (:3000) ── Hub ── Laptop 3 (:5000)
192.168.1.100              192.168.1.102
```

**Setup:**
```bash
# Laptop 1 (Hub)
./bin/fs -addr :3000 -interactive

# Laptop 2
./bin/fs -addr :4000 -bootstrap 192.168.1.100:3000 -interactive

# Laptop 3
./bin/fs -addr :5000 -bootstrap 192.168.1.100:3000 -interactive
```

### Mesh Network (3+ nodes)
```
Laptop 1 (:3000) ◄──► Laptop 2 (:4000)
192.168.1.100   ╲     ╱   192.168.1.101
                 ╲   ╱
                  ╲ ╱
            Laptop 3 (:5000)
             192.168.1.102
```

**Setup:**
```bash
# Laptop 1
./bin/fs -addr :3000 -interactive

# Laptop 2
./bin/fs -addr :4000 -bootstrap 192.168.1.100:3000 -interactive

# Laptop 3 (connects to both)
./bin/fs -addr :5000 -bootstrap 192.168.1.100:3000,192.168.1.101:4000 -interactive
```

## Troubleshooting

### Common Issues

**1. Connection Refused**
```
dial error: connect: connection refused
```
- Check if the target node is running
- Verify the IP address and port
- Check firewall settings

**2. Cannot Bind to Port**
```
listen tcp: bind: address already in use
```
- Port is already in use
- Choose a different port with `-addr :4000`

**3. No Peers Connected**
```
Connected peers: 0
```
- Check bootstrap addresses
- Ensure target nodes are running
- Verify network connectivity

**4. File Not Found**
```
Error retrieving file: need to serve file but it does not exist
```
- File may not be replicated to connected peers
- Try storing the file again
- Check network connectivity

### Firewall Configuration

**Windows Firewall:**
1. Open Windows Defender Firewall
2. Click "Allow an app through firewall"
3. Add `fs.exe` and allow both Private and Public networks

**Linux (ufw):**
```bash
sudo ufw allow 3000/tcp
sudo ufw allow 4000/tcp
sudo ufw allow 5000/tcp
```

**macOS:**
1. System Preferences → Security & Privacy → Firewall
2. Add PeerVault application to allowed apps

## Advanced Configuration

### Custom Network Key
For production use, you should use a custom network key. Modify `main.go`:

```go
// Replace this line in main.go:
networkKey := []byte("PeerVault-Network-Key-256bit!")

// With your custom 32-byte key:
networkKey := []byte("YourCustom32ByteKeyHere123456789")
```

### Port Range Setup
If running multiple nodes on the same machine:

```bash
# Terminal 1
./bin/fs -addr :3000 -interactive

# Terminal 2
./bin/fs -addr :3001 -bootstrap localhost:3000 -interactive

# Terminal 3
./bin/fs -addr :3002 -bootstrap localhost:3000,localhost:3001 -interactive
```

## Example Session

Here's a complete example of setting up a 3-node network:

**Laptop 1 (192.168.1.100):**
```bash
$ ./bin/fs -addr :3000 -interactive
Starting PeerVault server on :3000
Local IP: 192.168.1.100

PeerVault> store important.doc
File 'important.doc' stored successfully

PeerVault> status
Server listening on: :3000
Local IP: 192.168.1.100
Connected peers: 2
  - 192.168.1.101:4000
  - 192.168.1.102:5000
```

**Laptop 2 (192.168.1.101):**
```bash
$ ./bin/fs -addr :4000 -bootstrap 192.168.1.100:3000 -interactive
Starting PeerVault server on :4000
Local IP: 192.168.1.101

PeerVault> get important.doc
File content: Sample data for file: important.doc (stored at 14:30:25)

PeerVault> store backup.txt
File 'backup.txt' stored successfully
```

**Laptop 3 (192.168.1.102):**
```bash
$ ./bin/fs -addr :5000 -bootstrap 192.168.1.100:3000,192.168.1.101:4000 -interactive
Starting PeerVault server on :5000
Local IP: 192.168.1.102

PeerVault> get important.doc
File content: Sample data for file: important.doc (stored at 14:30:25)

PeerVault> get backup.txt
File content: Sample data for file: backup.txt (stored at 14:32:10)
```

This demonstrates a fully functional distributed file system across three laptops!

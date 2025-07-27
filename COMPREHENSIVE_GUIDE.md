# PeerVault: Comprehensive Project Guide

## Table of Contents
1. [Project Overview](#project-overview)
2. [Architecture & Design](#architecture--design)
3. [Core Components](#core-components)
4. [Technical Implementation](#technical-implementation)
5. [Network Protocol](#network-protocol)
6. [Security & Encryption](#security--encryption)
7. [File Storage System](#file-storage-system)
8. [Peer-to-Peer Communication](#peer-to-peer-communication)
9. [Development Setup](#development-setup)
10. [Usage Examples](#usage-examples)
11. [Testing Strategy](#testing-strategy)
12. [Project Structure](#project-structure)
13. [Future Enhancements](#future-enhancements)

---

## Project Overview

**PeerVault** is a sophisticated **Peer-to-Peer (P2P) distributed file storage system** built in Go that enables secure, redundant file storage across a network of connected nodes. The system combines distributed computing principles with modern cryptography to create a robust, decentralized file storage solution.

### Key Features
- **Distributed Architecture**: Files are replicated across multiple nodes for redundancy
- **End-to-End Encryption**: All files are encrypted using AES-256 before storage and transmission
- **Content-Addressed Storage (CAS)**: Files are stored using cryptographic hashes for deduplication
- **Custom P2P Protocol**: Lightweight TCP-based communication protocol
- **Streaming Support**: Efficient handling of large files through streaming
- **Automatic Peer Discovery**: Nodes can bootstrap and connect to existing network peers
- **Fault Tolerance**: Files remain accessible even if individual nodes go offline

### What Problem Does It Solve?
1. **Single Point of Failure**: Traditional centralized storage systems fail when the server goes down
2. **Data Security**: Files are encrypted and distributed, making unauthorized access extremely difficult
3. **Storage Costs**: Utilizes unused storage capacity across multiple machines
4. **Data Redundancy**: Automatic replication ensures data availability
5. **Privacy**: No central authority has access to your files

---

## Architecture & Design

### High-Level Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Node A        │    │   Node B        │    │   Node C        │
│  :3000          │◄──►│  :7000          │◄──►│  :5000          │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │FileServer   │ │    │ │FileServer   │ │    │ │FileServer   │ │
│ │- Store      │ │    │ │- Store      │ │    │ │- Store      │ │
│ │- Transport  │ │    │ │- Transport  │ │    │ │- Transport  │ │  
│ │- Peers      │ │    │ │- Peers      │ │    │ │- Peers      │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Design Principles
1. **Decentralization**: No single point of control or failure
2. **Modularity**: Clean separation between transport, storage, and application layers
3. **Security-First**: Encryption is built into the core architecture
4. **Scalability**: Can handle an arbitrary number of nodes
5. **Simplicity**: Minimal dependencies and straightforward protocol

---

## Core Components

### 1. FileServer (`server.go`)
The central orchestrator that manages all operations:

**Key Responsibilities:**
- **Peer Management**: Maintains connections to other nodes
- **Message Routing**: Handles incoming/outgoing messages
- **File Operations**: Coordinates storage and retrieval
- **Network Bootstrap**: Connects to existing network nodes

**Core Methods:**
- `Store(key, reader)`: Stores a file and replicates to peers
- `Get(key)`: Retrieves a file from local storage or network
- `broadcast(message)`: Sends messages to all connected peers
- `OnPeer(peer)`: Handles new peer connections

### 2. Storage System (`store.go`)
Manages local file storage with content-addressed system:

**Features:**
- **Path Transformation**: Converts keys to directory structures using SHA-1 hashing
- **Hierarchical Storage**: Organizes files in nested directories
- **Stream Support**: Handles large files efficiently
- **Encryption Integration**: Works seamlessly with crypto functions

**Storage Structure:**
```
storage_root/
├── node_id_1/
│   └── a1b2c/d3e4f/g5h6i/.../<full_hash>
├── node_id_2/
│   └── x1y2z/a3b4c/d5e6f/.../<full_hash>
```

### 3. P2P Transport Layer (`p2p/`)
Handles all network communication:

**Components:**
- **Transport Interface**: Defines communication contract
- **TCP Implementation**: Reliable TCP-based communication
- **Message Protocol**: Binary protocol for efficient communication
- **Peer Management**: Connection lifecycle management

### 4. Cryptography (`crypto.go`)
Provides security through encryption:

**Functions:**
- **Key Generation**: Creates cryptographically secure keys
- **AES Encryption**: Uses AES-256 in CTR mode
- **Streaming Crypto**: Encrypts/decrypts data streams
- **Hash Functions**: MD5 for keys, SHA-1 for content addressing

---

## Technical Implementation

### Message Types & Protocol

The system uses a binary protocol with two main message types:

#### 1. MessageStoreFile
```go
type MessageStoreFile struct {
    ID   string  // Node identifier
    Key  string  // File key (hashed)
    Size int64   // File size in bytes
}
```
**Purpose**: Notifies peers that a file is being stored
**Flow**: Store → Broadcast → Peer receives → Peer stores locally

#### 2. MessageGetFile
```go
type MessageGetFile struct {
    ID  string  // Requesting node ID
    Key string  // Requested file key
}
```
**Purpose**: Requests a file from the network
**Flow**: Get → Broadcast → Peer responds → File transferred

### Communication Flow

#### File Storage Process:
1. **Local Storage**: File encrypted and stored locally
2. **Broadcast Notification**: `MessageStoreFile` sent to all peers
3. **Peer Preparation**: Peers prepare to receive file
4. **Stream Transfer**: Encrypted file streamed to all peers
5. **Confirmation**: Peers store file locally

#### File Retrieval Process:
1. **Local Check**: Check if file exists locally
2. **Network Request**: If not found, broadcast `MessageGetFile`
3. **Peer Response**: Peer with file responds with stream
4. **Local Storage**: Received file decrypted and stored locally
5. **Return**: File reader returned to caller

---

## Network Protocol

### Protocol Layers
```
┌─────────────────────┐
│   Application       │ ← File operations (Store/Get)
├─────────────────────┤
│   Message Layer     │ ← MessageStoreFile, MessageGetFile
├─────────────────────┤
│   RPC Layer         │ ← Remote Procedure Calls
├─────────────────────┤
│   Transport Layer   │ ← TCP connections
└─────────────────────┘
```

### Message Format
- **Header Byte**: Indicates message type (0x1 = Message, 0x2 = Stream)
- **Payload**: GOB-encoded message data
- **Stream Data**: Raw binary data for file transfers

### Connection Management
- **Persistent Connections**: Maintains long-lived TCP connections
- **Handshake Process**: Verifies peer identity (currently NOP for development)
- **Graceful Shutdown**: Proper connection cleanup
- **Error Handling**: Robust error recovery and connection retry

---

## Security & Encryption

### Encryption Strategy
**Algorithm**: AES-256 in CTR (Counter) mode
**Key Management**: Per-node random 256-bit keys
**IV Generation**: Random initialization vector per file

### Security Features
1. **Data at Rest**: All files encrypted before storage
2. **Data in Transit**: Files encrypted during network transfer  
3. **Content Addressing**: SHA-1 hashes prevent tampering
4. **Node Isolation**: Each node has isolated storage
5. **Key Rotation**: Keys can be regenerated per session

### Cryptographic Functions
```go
// Encryption flow
copyEncrypt(key, plaintext, ciphertext) -> encrypted_data + IV

// Decryption flow  
copyDecrypt(key, ciphertext_with_IV, plaintext) -> original_data
```

### Security Considerations
- **Key Storage**: Keys stored in memory (not persisted)
- **Perfect Forward Secrecy**: New keys per session
- **Side-Channel Resistance**: Uses cryptographically secure random number generator
- **Integrity**: Content addressing provides integrity verification

---

## File Storage System

### Content-Addressed Storage (CAS)
Files are stored using their content hash, providing:
- **Deduplication**: Identical files stored only once
- **Integrity**: Content verification through hashing
- **Immutable Storage**: Content cannot be modified without changing hash

### Path Transformation Algorithm
```go
// Example: key = "myfile.txt"
// 1. Hash: SHA1("myfile.txt") = "a1b2c3d4e5f6..."
// 2. Split: ["a1b2c", "3d4e5", "f6789", ...]
// 3. Path: "a1b2c/3d4e5/f6789/.../a1b2c3d4e5f6..."
```

### Storage Benefits
- **Balanced Distribution**: Even distribution across directory structure
- **Scalability**: Handles millions of files efficiently
- **Fast Lookup**: O(1) file access time
- **Collision Resistance**: SHA-1 provides strong collision resistance

### File Operations
- **Write**: Stream → Encrypt → Store → Notify peers
- **Read**: Locate → Decrypt → Stream to caller
- **Delete**: Remove file and containing directories
- **Has**: Check file existence without reading

---

## Peer-to-Peer Communication

### Network Topology
- **Mesh Network**: Each node can connect to multiple peers
- **Bootstrap Nodes**: Initial connection points for network discovery
- **Dynamic Connections**: Nodes can join/leave network dynamically

### Peer Discovery Process
1. **Bootstrap**: Connect to known nodes specified in configuration
2. **Peer Exchange**: Learn about other peers through connected nodes
3. **Connection Establishment**: Establish TCP connections to new peers
4. **Handshake**: Verify peer identity and capabilities

### Message Broadcasting
- **Flood Protocol**: Messages sent to all connected peers
- **Reliability**: TCP ensures message delivery
- **Ordering**: Messages processed in arrival order
- **Deduplication**: Nodes ignore duplicate messages

### Connection Management
```go
type TCPPeer struct {
    net.Conn                // Underlying TCP connection
    outbound bool          // Connection direction
    wg       *sync.WaitGroup // Stream synchronization
}
```

---

## Development Setup

### Prerequisites
- **Go 1.22.5+**: Modern Go version with generics support
- **Git**: For version control
- **Make**: For build automation
- **Network Access**: For P2P communication

### Installation Steps
```bash
# 1. Clone repository
git clone https://github.com/AdityaKrSingh26/PeerVault.git
cd PeerVault

# 2. Install dependencies
go mod tidy

# 3. Build project
make build

# 4. Run tests
make test
```

### Build Targets
- `make build`: Compiles binary to `bin/fs`
- `make run`: Builds and runs with default settings
- `make test`: Runs all unit tests

### Development Tools
- **Testing Framework**: Uses `testify` for assertions
- **Build System**: Make-based build automation
- **Module System**: Go modules for dependency management

---

## Usage Examples

### Basic Network Setup
```bash
# Terminal 1: Start first node
./bin/fs -addr :3000

# Terminal 2: Start second node  
./bin/fs -addr :7000

# Terminal 3: Start third node with bootstrap
./bin/fs -addr :5000 -bootstrap :3000,:7000
```

### Programmatic Usage
```go
// Create file server
server := makeServer(":3000", ":7000", ":5000")

// Store a file
data := bytes.NewReader([]byte("Hello PeerVault!"))
err := server.Store("myfile.txt", data)

// Retrieve a file
reader, err := server.Get("myfile.txt")
content, _ := ioutil.ReadAll(reader)
fmt.Println(string(content)) // "Hello PeerVault!"
```

### Advanced Configuration
```go
// Custom file server options
opts := FileServerOpts{
    ID:                "custom-node-id",
    EncKey:           customEncryptionKey,
    StorageRoot:      "/custom/storage/path",
    PathTransformFunc: CASPathTransformFunc,
    Transport:        tcpTransport,
    BootstrapNodes:   []string{":3000", ":7000"},
}
```

---

## Testing Strategy

### Unit Tests
- **Store Tests**: File storage and retrieval operations
- **Crypto Tests**: Encryption/decryption functionality
- **Transport Tests**: Network communication
- **Integration Tests**: End-to-end file operations

### Test Coverage Areas
1. **Storage Operations**: Write, read, delete, exists checks
2. **Cryptographic Functions**: Encryption/decryption correctness
3. **Network Protocol**: Message serialization/deserialization
4. **Peer Management**: Connection handling and lifecycle
5. **Error Scenarios**: Network failures, corrupted data, missing files

### Running Tests
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./p2p
go test -v .  # Verbose output

# Run with coverage
go test -cover ./...
```

### Test Structure
```go
func TestStoreHas(t *testing.T) {
    // Setup
    store := NewStore(StoreOpts{})
    
    // Test cases
    assert.False(t, store.Has("id", "nonexistent"))
    
    // Store file and verify
    store.Write("id", "key", reader)
    assert.True(t, store.Has("id", "key"))
}
```

---

## Project Structure

### Directory Layout
```
PeerVault/
├── main.go              # Entry point and demo
├── server.go            # Core FileServer implementation
├── store.go             # File storage system
├── crypto.go            # Cryptographic functions
├── go.mod              # Go module definition
├── go.sum              # Dependency checksums
├── Makefile            # Build automation
├── README.md           # Basic project documentation
├── *_test.go           # Unit tests
├── bin/                # Compiled binaries
│   └── fs.exe          # Main executable
└── p2p/                # P2P networking package
    ├── transport.go    # Transport interface
    ├── tcp_transport.go # TCP implementation
    ├── message.go      # Message definitions
    ├── encoding.go     # Message serialization
    ├── handshake.go    # Peer handshake
    └── *_test.go       # P2P tests
```

### Code Organization
- **Main Package**: Core application logic
- **p2p Package**: Network communication abstractions
- **Interfaces**: Clean abstractions for testability
- **Configuration**: Struct-based configuration pattern
- **Error Handling**: Explicit error returns

### Dependencies
```go
require (
    github.com/stretchr/testify v1.10.0  // Testing framework
)
```

---

## Future Enhancements

### Planned Features
1. **Enhanced Security**
   - Digital signatures for message authentication
   - Certificate-based peer verification
   - Key rotation and forward secrecy

2. **Network Improvements**
   - DHT (Distributed Hash Table) for better peer discovery
   - NAT traversal for nodes behind firewalls
   - Connection pooling and load balancing

3. **Storage Optimizations**
   - Compression before encryption
   - Erasure coding for space efficiency
   - Garbage collection for unused files

4. **User Interface**
   - Web-based management interface
   - REST API for external integrations
   - Command-line tools for file operations

5. **Monitoring & Analytics**
   - Metrics collection and reporting
   - Health monitoring and alerts
   - Performance profiling tools

### Potential Use Cases
- **Personal Cloud Storage**: Distributed backup across personal devices
- **Team Collaboration**: Secure file sharing within organizations
- **Content Distribution**: Decentralized content delivery network
- **Archive Storage**: Long-term storage with redundancy
- **IoT Data Storage**: Distributed storage for IoT sensor data

### Technical Debt & Improvements
1. **Error Handling**: More granular error types and recovery
2. **Configuration**: YAML/JSON-based configuration files
3. **Logging**: Structured logging with levels and rotation
4. **Documentation**: API documentation and tutorials
5. **Performance**: Benchmarking and optimization

---

## Conclusion

PeerVault represents a robust implementation of distributed file storage principles, combining modern cryptography with peer-to-peer networking. The system demonstrates how to build scalable, secure, and fault-tolerant distributed applications using Go.

The project showcases several important concepts:
- **Distributed Systems Design**: How to architect systems without central points of failure
- **Cryptographic Integration**: Proper use of encryption in distributed systems
- **Network Programming**: Building custom protocols for P2P communication
- **Storage Systems**: Content-addressed storage and path transformation
- **Go Concurrency**: Effective use of goroutines and channels

Whether used as a learning resource, a foundation for larger projects, or adapted for specific use cases, PeerVault provides a solid starting point for understanding and implementing distributed storage systems.

The clean architecture, comprehensive testing, and well-documented code make it an excellent reference implementation for anyone interested in distributed systems, P2P networking, or secure file storage solutions.

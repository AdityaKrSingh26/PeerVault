# PeerVault

PeerVault is a **P2P distributed file storage system** that allows you to store and retrieve files across a distributed network of nodes. It uses encryption to secure your files and ensures redundancy by broadcasting files to multiple peers in the network and makes sure they’re available even if one computer goes offline.

---

## What It Does

- **Distributed File Storage**: Files are stored across multiple nodes in the network.
- **Encryption**: Files are encrypted before being stored and decrypted when retrieved.
- **Streaming Support**: Handles large files efficiently using streaming.
- **Custom Protocol**: Uses a lightweight custom protocol for communication between nodes.
- **Peer Discovery**: Automatically connects to other nodes in the network for redundancy.

---

## Flow
![flow](https://github.com/user-attachments/assets/b409604d-eddf-4b39-8f9d-d098e96e0f07)

## How to Use

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

### 2. Run
Start a node:
```bash
./bin/fs -addr :3000
```

### 3. Store a File
Store a file like this:
```go
data := bytes.NewReader([]byte("Hello, PeerVault!"))
fileServer.Store("myfile.txt", data)
```

### 4. Get a File
Retrieve a file like this:
```go
reader, err := fileServer.Get("myfile.txt")
if err != nil {
    log.Fatal(err)
}
data, _ := ioutil.ReadAll(reader)
fmt.Println(string(data)) // Output: Hello, PeerVault!
```

---

## Example

1. Start the first computer:
   ```bash
   ./bin/fs -addr :3000
   ```

2. Start the second computer:
   ```bash
   ./bin/fs -addr :7000
   ```

3. Start the third computer and connect to the first two:
   ```bash
   ./bin/fs -addr :5000 -bootstrap :3000,:7000
   ```

Now, you can store and retrieve files across all three computers!

---

## Testing

Run tests to make sure everything works:
```bash
make test
```

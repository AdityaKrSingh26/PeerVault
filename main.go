package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/AdityaKrSingh26/PeerVault/p2p"
)

func makeServer(listenAddr string, networkKey []byte, nodes ...string) *FileServer {
	tcptransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcptransportOpts)

	// Create a safe storage root name in a dedicated storage directory
	// Replace : with _ for Windows compatibility
	portName := strings.ReplaceAll(listenAddr, ":", "port_")
	storageRoot := fmt.Sprintf("storage/node_%s", portName)

	fileServerOpts := FileServerOpts{
		EncKey:            networkKey, // Use shared network key
		StorageRoot:       storageRoot,
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer

	return s
}

// Get the local IP address
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// Interactive mode for file operations
func interactiveMode(server *FileServer) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\n=== PeerVault Interactive Mode ===")
	fmt.Println("Commands:")
	fmt.Println("  store <filename>  - Store a file with sample data")
	fmt.Println("  get <filename>    - Retrieve and display a file")
	fmt.Println("  list              - List all stored files")
	fmt.Println("  status            - Show server and network status")
	fmt.Println("  clean             - Clean local storage")
	fmt.Println("  quit              - Exit PeerVault")
	fmt.Println()

	for {
		fmt.Print("PeerVault> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Split(input, " ")
		command := parts[0]

		switch command {
		case "store":
			if len(parts) < 2 {
				fmt.Println("Usage: store <filename>")
				continue
			}
			filename := parts[1]
			// For demo, store some sample data
			data := bytes.NewReader([]byte(fmt.Sprintf("Sample data for file: %s (stored at %s)", filename, time.Now().Format("15:04:05"))))
			err := server.Store(filename, data)
			if err != nil {
				fmt.Printf("Error storing file: %v\n", err)
			} else {
				fmt.Printf("File '%s' stored successfully\n", filename)
			}

		case "get":
			if len(parts) < 2 {
				fmt.Println("Usage: get <filename>")
				continue
			}
			filename := parts[1]
			reader, err := server.Get(filename)
			if err != nil {
				fmt.Printf("Error retrieving file: %v\n", err)
			} else {
				data, err := io.ReadAll(reader)
				if err != nil {
					fmt.Printf("Error reading file: %v\n", err)
				} else {
					fmt.Printf("File content: %s\n", string(data))
				}
			}

		case "status":
			fmt.Printf("Server listening on: %s\n", server.Transport.Addr())
			fmt.Printf("Local IP: %s\n", getLocalIP())
			fmt.Printf("Connected peers: %d\n", len(server.peers))
			for addr := range server.peers {
				fmt.Printf("  - %s\n", addr)
			}

		case "list":
			// List files stored on this node
			files, err := server.store.List(server.ID)
			if err != nil {
				fmt.Printf("Error listing files: %v\n", err)
				continue
			}

			if len(files) == 0 {
				fmt.Println("No files stored on this node")
			} else {
				fmt.Printf("Files stored on this node (%d files):\n", len(files))
				fmt.Println("┌─────────────────────────────────────┬─────────────┬──────────────────────┐")
				fmt.Println("│ Filename                            │ Size (bytes)│ Hash (first 8 chars) │")
				fmt.Println("├─────────────────────────────────────┼─────────────┼──────────────────────┤")
				for _, file := range files {
					filename := file.Key
					if len(filename) > 35 {
						filename = filename[:32] + "..."
					}
					hashShort := file.Hash
					if len(hashShort) > 8 {
						hashShort = hashShort[:8]
					}
					fmt.Printf("│ %-35s │ %11d │ %-20s │\n", filename, file.Size, hashShort)
				}
				fmt.Println("└─────────────────────────────────────┴─────────────┴──────────────────────┘")
			}

			// Also show files from other nodes (if any)
			allFiles, err := server.store.ListAll()
			if err == nil && len(allFiles) > 1 {
				fmt.Println("\nFiles from other nodes:")
				for nodeID, nodeFiles := range allFiles {
					if nodeID != server.ID && len(nodeFiles) > 0 {
						fmt.Printf("  Node %s (%d files):\n", nodeID[:8], len(nodeFiles))
						for _, file := range nodeFiles {
							fmt.Printf("    - %s (%d bytes)\n", file.Key, file.Size)
						}
					}
				}
			}

		case "clean":
			fmt.Print("Are you sure you want to delete all local files? (y/N): ")
			if !scanner.Scan() {
				continue
			}
			confirmation := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if confirmation == "y" || confirmation == "yes" {
				// First stop the server to close any open files
				server.Stop()
				time.Sleep(500 * time.Millisecond) // Give time for cleanup

				err := server.store.Clear()
				if err != nil {
					fmt.Printf("Error cleaning storage: %v\n", err)
				} else {
					fmt.Println("Local storage cleaned successfully")
					// Clear the key mapping as well
					server.store.keyMap = make(map[string]string)
				}

				fmt.Println("Server stopped. Please restart to continue.")
				return
			} else {
				fmt.Println("Clean operation cancelled")
			}

		case "quit", "exit":
			fmt.Println("Shutting down...")
			server.Stop()
			return

		default:
			fmt.Printf("Unknown command: %s\n", command)
		}
	}
}

func main() {
	// Command line flags
	var (
		listenAddr  = flag.String("addr", ":3000", "Listen address (e.g., :3000)")
		bootstrap   = flag.String("bootstrap", "", "Bootstrap nodes (comma-separated, e.g., 192.168.1.100:3000,192.168.1.101:4000)")
		interactive = flag.Bool("interactive", false, "Run in interactive mode")
		demo        = flag.Bool("demo", false, "Run demo mode with test data")
	)
	flag.Parse()

	// Generate a shared network key (in production, this should be provided via config)
	// For now, using a deterministic key so all nodes can decrypt each other's files
	networkKey := []byte("PeerVault-Network-Key-256bit!") // 32 bytes for AES-256
	if len(networkKey) != 32 {
		// Pad or truncate to 32 bytes
		key := make([]byte, 32)
		copy(key, networkKey)
		networkKey = key
	}

	// Parse bootstrap nodes
	var bootstrapNodes []string
	if *bootstrap != "" {
		bootstrapNodes = strings.Split(*bootstrap, ",")
		// Trim whitespace
		for i, node := range bootstrapNodes {
			bootstrapNodes[i] = strings.TrimSpace(node)
		}
	}

	// Create and start server
	server := makeServer(*listenAddr, networkKey, bootstrapNodes...)

	// Start server in background
	go func() {
		log.Printf("Starting PeerVault server on %s", *listenAddr)
		log.Printf("Local IP: %s", getLocalIP())
		if len(bootstrapNodes) > 0 {
			log.Printf("Bootstrap nodes: %v", bootstrapNodes)
		}

		if err := server.Start(); err != nil {
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Give server time to start
	time.Sleep(2 * time.Second)

	if *interactive {
		// Interactive mode
		interactiveMode(server)
	} else if *demo {
		// Demo mode - store and retrieve some test files
		fmt.Println("Running demo mode...")

		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("demo_file_%d.txt", i)
			data := bytes.NewReader([]byte(fmt.Sprintf("Demo file %d content created at %s", i, time.Now().Format("15:04:05"))))

			if err := server.Store(key, data); err != nil {
				log.Printf("Error storing %s: %v", key, err)
			} else {
				log.Printf("Stored: %s", key)
			}
		}

		time.Sleep(2 * time.Second)

		// Try to retrieve files
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("demo_file_%d.txt", i)
			reader, err := server.Get(key)
			if err != nil {
				log.Printf("Error retrieving %s: %v", key, err)
			} else {
				data, _ := io.ReadAll(reader)
				log.Printf("Retrieved %s: %s", key, string(data))
			}
		}
	} else {
		// Keep server running
		fmt.Printf("PeerVault server running on %s\n", *listenAddr)
		fmt.Printf("Local IP: %s\n", getLocalIP())
		fmt.Printf("Use Ctrl+C to stop or --interactive flag for interactive mode\n")

		select {} // Block forever
	}
}

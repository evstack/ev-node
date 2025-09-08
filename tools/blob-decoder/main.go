package main

import (
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

//go:embed templates/*
var templatesFS embed.FS

// DecodedBlob represents the result of decoding a blob
type DecodedBlob struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	RawHex    string      `json:"rawHex"`
	Size      int         `json:"size"`
	Timestamp time.Time   `json:"timestamp"`
	Error     string      `json:"error,omitempty"`
}

func main() {
	mux := http.NewServeMux()

	// handlers
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/decode", handleDecode)

	// port configuration
	port := "8090"
	if len(os.Args[1:]) > 0 {
		port = os.Args[1]
		if _, err := strconv.Atoi(port); err != nil {
			log.Fatal("Invalid port number")
		}
	}

	fmt.Printf(`
╔═══════════════════════════════════╗
║        Evolve Blob Decoder        ║
║           by ev.xyz               ║
╚═══════════════════════════════════╝

🚀 Server running at: http://localhost:%s
⚡ Using native Evolve protobuf decoding

Press Ctrl+C to stop the server
`, port)

	// Create server with proper timeout configuration
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/index.html"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleDecode(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Data     string `json:"data"`
		Encoding string `json:"encoding"` // "hex" or "base64"
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Decode the input data
	var blobData []byte
	var err error

	switch request.Encoding {
	case "hex":
		cleanHex := strings.TrimPrefix(request.Data, "0x")
		blobData, err = hex.DecodeString(cleanHex)
	case "base64":
		blobData, err = base64.StdEncoding.DecodeString(request.Data)
	default:
		http.Error(w, "Invalid encoding type", http.StatusBadRequest)
		return
	}

	if err != nil {
		sendJSONResponse(w, DecodedBlob{
			Error:     fmt.Sprintf("Failed to decode %s: %v", request.Encoding, err),
			Timestamp: time.Now(),
		})
		return
	}

	// Try to decode the blob
	result := decodeBlob(blobData)
	result.RawHex = hex.EncodeToString(blobData)
	result.Size = len(blobData)
	result.Timestamp = time.Now()

	sendJSONResponse(w, result)
}

func decodeBlob(data []byte) DecodedBlob {
	// Try to decode as SignedHeader
	if header := tryDecodeHeader(data); header != nil {
		return DecodedBlob{
			Type: "SignedHeader",
			Data: header,
		}
	}

	// Try to decode as SignedData
	if signedData := tryDecodeSignedData(data); signedData != nil {
		return DecodedBlob{
			Type: "SignedData",
			Data: signedData,
		}
	}

	// Check if it's JSON
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		return DecodedBlob{
			Type: "JSON",
			Data: jsonData,
		}
	}

	// Return as unknown binary
	return DecodedBlob{
		Type: "Unknown",
		Data: map[string]interface{}{
			"message": "Unable to decode blob format",
			"preview": hex.EncodeToString(data[:min(100, len(data))]),
		},
	}
}

func tryDecodeHeader(data []byte) interface{} {
	var headerPb pb.SignedHeader
	if err := proto.Unmarshal(data, &headerPb); err != nil {
		return nil
	}

	var signedHeader types.SignedHeader
	if err := signedHeader.FromProto(&headerPb); err != nil {
		return nil
	}

	// Basic validation
	if err := signedHeader.Header.ValidateBasic(); err != nil {
		return nil
	}

	// Return a map with the actual header fields
	return map[string]interface{}{
		// BaseHeader fields
		"height":  signedHeader.Height(),
		"time":    signedHeader.Time().Format(time.RFC3339Nano),
		"chainId": signedHeader.ChainID(),

		// Version
		"version": map[string]interface{}{
			"block": signedHeader.Version.Block,
			"app":   signedHeader.Version.App,
		},

		// Hash fields (convert [32]byte arrays to hex strings)
		"lastHeaderHash":  bytesToHex(signedHeader.LastHeaderHash[:]),
		"lastCommitHash":  bytesToHex(signedHeader.LastCommitHash[:]),
		"dataHash":        bytesToHex(signedHeader.DataHash[:]),
		"consensusHash":   bytesToHex(signedHeader.ConsensusHash[:]),
		"appHash":         bytesToHex(signedHeader.AppHash[:]),
		"lastResultsHash": bytesToHex(signedHeader.LastResultsHash[:]),
		"validatorHash":   bytesToHex(signedHeader.ValidatorHash[:]),

		// Proposer
		"proposerAddress": bytesToHex(signedHeader.ProposerAddress),

		// Signature fields
		"signature": bytesToHex(signedHeader.Signature),
		"signer": map[string]interface{}{
			"address": bytesToHex(signedHeader.Signer.Address),
			"pubKey":  "",
		},
	}
}

func tryDecodeSignedData(data []byte) interface{} {
	var signedData types.SignedData
	if err := signedData.UnmarshalBinary(data); err != nil {
		return nil
	}

	// Create transaction list
	transactions := make([]map[string]interface{}, len(signedData.Txs))
	for i, tx := range signedData.Txs {
		transactions[i] = map[string]interface{}{
			"index": i,
			"size":  len(tx),
			"data":  bytesToHex(tx),
		}
	}

	result := map[string]interface{}{
		"transactions":     transactions,
		"transactionCount": len(signedData.Txs),
		"signature":        bytesToHex(signedData.Signature),
	}

	// Add Metadata fields if present
	if signedData.Metadata != nil {
		result["chainId"] = signedData.ChainID()
		result["height"] = signedData.Height()
		result["time"] = signedData.Time().Format(time.RFC3339Nano)
		result["lastDataHash"] = bytesToHex(signedData.LastDataHash[:])
	}

	// Add DACommitment hash
	dataHash := signedData.DACommitment()
	result["dataHash"] = bytesToHex(dataHash[:])

	return result
}

func bytesToHex(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return hex.EncodeToString(b)
}

func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

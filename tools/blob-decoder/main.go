package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// Removed embed since we're serving HTML inline

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

	// Serve the main page
	mux.HandleFunc("/", handleIndex)

	// API endpoint for decoding blobs
	mux.HandleFunc("/api/decode", handleDecode)

	// Static files removed - HTML is served inline

	port := "8090"
	fmt.Printf(`
╔════════════════════════════════════════════╗
║        Evolve Blob Decoder                ║
║           by ev.xyz                        ║
╚════════════════════════════════════════════╝

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

	log.Fatal(srv.ListenAndServe())
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
	tmpl := template.Must(template.New("index").Parse(indexHTML))
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Evolve Blob Decoder | ev.xyz</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        :root {
            /* Evolve brand colors from ev.xyz */
            --ev-brand: #ff0033;
            --ev-brand-light: #ff3355;
            --ev-brand-lighter: #ff6677;
            --ev-brand-lightest: #ffccdd;
            --ev-brand-dark: #cc0029;
            --ev-brand-darker: #990020;
            --ev-brand-dimm: rgba(255, 0, 51, 0.08);
            --ev-violet: #4c1d95;
            --ev-violet-light: #6d28d9;

            /* Dark mode (default) */
            --ev-bg: #0a0a0a;
            --ev-bg-soft: #141414;
            --ev-bg-mute: #1a1a1a;
            --ev-slate: #475569;
            --ev-text: #e5e7eb;
            --ev-text-secondary: #94a3b8;
            --ev-border: var(--ev-slate);
        }

        /* Light mode */
        body.light-mode {
            --ev-bg: #ffffff;
            --ev-bg-soft: #f9fafb;
            --ev-bg-mute: #f3f4f6;
            --ev-slate: #64748b;
            --ev-text: #1f2937;
            --ev-text-secondary: #6b7280;
            --ev-border: #e5e7eb;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: var(--ev-bg);
            min-height: 100vh;
            padding: 20px;
            position: relative;
            overflow-x: hidden;
            transition: background-color 0.3s ease, color 0.3s ease;
        }

        /* Background pattern with red squares */
        body::before {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            pointer-events: none;
            background-image:
                linear-gradient(rgba(255, 0, 51, 0.15) 2px, transparent 2px),
                linear-gradient(90deg, rgba(255, 0, 51, 0.15) 2px, transparent 2px);
            background-size: 60px 60px;
            z-index: 0;
        }

        /* Diagonal pattern overlay */
        body::after {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            pointer-events: none;
            background:
                repeating-linear-gradient(
                    45deg,
                    transparent,
                    transparent 35px,
                    rgba(255, 0, 51, 0.03) 35px,
                    rgba(255, 0, 51, 0.03) 70px
                );
            z-index: 0;
        }

        body.light-mode::before {
            background-image:
                linear-gradient(rgba(255, 0, 51, 0.1) 2px, transparent 2px),
                linear-gradient(90deg, rgba(255, 0, 51, 0.1) 2px, transparent 2px);
        }

        body.light-mode::after {
            background:
                repeating-linear-gradient(
                    45deg,
                    transparent,
                    transparent 35px,
                    rgba(255, 0, 51, 0.02) 35px,
                    rgba(255, 0, 51, 0.02) 70px
                );
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: var(--ev-bg-soft);
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(255,0,51,0.1);
            overflow: hidden;
            border: 1px solid var(--ev-bg-mute);
            position: relative;
            z-index: 1;
        }

        header {
            background: linear-gradient(135deg, var(--ev-brand) 30%, var(--ev-violet-light));
            color: white;
            padding: 30px;
            text-align: center;
            position: relative;
        }

        .brand {
            position: absolute;
            top: 15px;
            right: 20px;
            font-size: 14px;
            opacity: 0.9;
            font-weight: 500;
        }

        /* Theme toggle button */
        .theme-toggle {
            position: absolute;
            top: 15px;
            left: 20px;
            background: rgba(255, 255, 255, 0.2);
            border: 1px solid rgba(255, 255, 255, 0.3);
            border-radius: 20px;
            padding: 8px 16px;
            cursor: pointer;
            transition: all 0.3s;
            color: white;
            font-size: 14px;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .theme-toggle:hover {
            background: rgba(255, 255, 255, 0.3);
        }

        header h1 {
            font-size: 2.5rem;
            margin-bottom: 10px;
        }

        .content {
            padding: 30px;
            color: var(--ev-text);
        }

        .input-section {
            margin-bottom: 30px;
        }

        .input-section h2 {
            color: var(--ev-brand-light);
            margin-bottom: 20px;
        }

        .tabs {
            display: flex;
            gap: 10px;
            margin-bottom: 20px;
        }

        .tab {
            padding: 10px 20px;
            background: var(--ev-bg-mute);
            border: 1px solid var(--ev-slate);
            border-radius: 8px;
            cursor: pointer;
            font-size: 14px;
            transition: all 0.3s;
            color: var(--ev-text);
        }

        .tab.active {
            background: var(--ev-brand);
            color: white;
            border-color: var(--ev-brand);
        }

        textarea {
            width: 100%;
            padding: 15px;
            border: 2px solid var(--ev-slate);
            border-radius: 8px;
            font-family: 'Monaco', monospace;
            font-size: 14px;
            resize: vertical;
            min-height: 200px;
            background: var(--ev-bg);
            color: var(--ev-text);
        }

        textarea:focus {
            outline: none;
            border-color: var(--ev-brand);
            box-shadow: 0 0 0 3px var(--ev-brand-dimm);
        }

        .decode-btn {
            width: 100%;
            padding: 15px;
            background: linear-gradient(135deg, var(--ev-brand), var(--ev-violet-light));
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            margin-top: 15px;
            transition: all 0.3s;
        }

        .decode-btn:hover:not(:disabled) {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(0,0,0,0.2);
        }

        .decode-btn:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }

        .result-section {
            margin-top: 30px;
            display: none;
        }

        .result-section.active {
            display: block;
        }

        .result-header {
            display: flex;
            justify-content: space-between;
            padding: 15px 20px;
            background: linear-gradient(135deg, var(--ev-brand), var(--ev-brand-dark));
            color: white;
            border-radius: 8px 8px 0 0;
        }

        .result-content {
            background: var(--ev-bg);
            border: 1px solid var(--ev-bg-mute);
            border-radius: 0 0 8px 8px;
            padding: 20px;
            max-height: 600px;
            overflow-y: auto;
        }

        .field {
            display: flex;
            padding: 12px 0;
            border-bottom: 1px solid var(--ev-bg-mute);
        }

        .field-name {
            font-weight: 600;
            color: var(--ev-brand-light);
            min-width: 180px;
        }

        .field-value {
            color: var(--ev-text);
            word-break: break-all;
            font-family: 'Monaco', monospace;
            font-size: 13px;
        }

        .field-value.hash {
            background: var(--ev-bg-mute);
            padding: 4px 8px;
            border-radius: 4px;
            color: var(--ev-text-secondary);
        }

        .error {
            background: var(--ev-brand-dimm);
            border: 1px solid var(--ev-brand);
            color: var(--ev-brand-light);
            padding: 15px;
            border-radius: 8px;
            margin-top: 15px;
        }

        .loading {
            text-align: center;
            padding: 40px;
            color: var(--ev-slate);
        }

        .spinner {
            border: 3px solid var(--ev-bg-mute);
            border-top: 3px solid var(--ev-brand);
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 20px;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        .tx-list {
            margin-top: 20px;
        }

        .tx-item {
            background: var(--ev-bg-mute);
            padding: 10px;
            margin-bottom: 10px;
            border-radius: 6px;
            border: 1px solid var(--ev-slate);
        }

        .tx-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 5px;
            font-weight: 600;
            color: var(--ev-brand-light);
        }

        .tx-hash {
            font-family: 'Monaco', monospace;
            font-size: 12px;
            color: var(--ev-slate);
        }

        pre {
            background: var(--ev-bg);
            color: var(--ev-text);
            padding: 15px;
            border-radius: 8px;
            overflow-x: auto;
            font-size: 13px;
            line-height: 1.5;
            border: 1px solid var(--ev-bg-mute);
        }

        .view-tabs {
            display: flex;
            gap: 10px;
            margin-bottom: 15px;
        }

        .view-tab {
            padding: 8px 16px;
            background: var(--ev-bg-mute);
            border: 1px solid var(--ev-slate);
            border-radius: 6px;
            cursor: pointer;
            font-size: 14px;
            transition: all 0.3s;
            color: var(--ev-text);
        }

        .view-tab.active {
            background: var(--ev-brand);
            color: white;
            border-color: var(--ev-brand);
        }

        .view-content {
            display: none;
        }

        .view-content.active {
            display: block;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <button class="theme-toggle" onclick="toggleTheme()">
                <span id="themeIcon">🌙</span>
                <span id="themeText">Dark</span>
            </button>
            <div class="brand">ev.xyz</div>
            <h1>⚡ Evolve Blob Decoder</h1>
            <p>Own It. Shape It. Launch It.</p>
            <p style="font-size: 14px; margin-top: 8px; opacity: 0.9;">Decode and inspect DA layer blobs from your Evolve rollup</p>
        </header>

        <div class="content">
            <div class="input-section">
                <h2>Input Blob Data</h2>
                <div class="tabs">
                    <button class="tab active" data-encoding="hex">Hex String</button>
                    <button class="tab" data-encoding="base64">Base64</button>
                </div>

                <textarea id="blobInput" placeholder="Paste your hex or base64 encoded blob here..."></textarea>
                <button class="decode-btn" onclick="decodeBlob()">Decode Blob</button>
            </div>

            <div id="loading" class="loading" style="display: none;">
                <div class="spinner"></div>
                <p>Decoding blob...</p>
            </div>

            <div id="error" class="error" style="display: none;"></div>

            <div id="result" class="result-section">
                <div class="result-header">
                    <span id="blobType"></span>
                    <span id="blobSize"></span>
                </div>

                <div class="result-content">
                    <div class="view-tabs">
                        <button class="view-tab active" data-view="parsed">Parsed</button>
                        <button class="view-tab" data-view="raw">Raw JSON</button>
                        <button class="view-tab" data-view="hex">Hex Dump</button>
                    </div>

                    <div id="parsedView" class="view-content active"></div>
                    <div id="rawView" class="view-content"></div>
                    <div id="hexView" class="view-content"></div>
                </div>
            </div>
        </div>

        <footer style="text-align: center; padding: 20px; color: #6b7280; font-size: 14px;">
            <div style="margin-bottom: 10px;">
                <strong style="color: var(--ev-brand);">Evolve</strong> - The Modular Rollup Framework
            </div>
            <div>
                🚀 Full Control Over Execution &nbsp;|&nbsp;
                ⚡ Speed to Traction &nbsp;|&nbsp;
                🛡️ No Validator Overhead
            </div>
            <div style="margin-top: 10px;">
                <a href="https://ev.xyz" target="_blank" style="color: var(--ev-brand-light); text-decoration: none;">ev.xyz</a> &nbsp;|&nbsp;
                <a href="https://docs.ev.xyz" target="_blank" style="color: var(--ev-brand-light); text-decoration: none;">Documentation</a> &nbsp;|&nbsp;
                <a href="https://github.com/evstack" target="_blank" style="color: var(--ev-brand-light); text-decoration: none;">GitHub</a>
            </div>
        </footer>
    </div>

    <script>
        let currentEncoding = 'hex';
        let currentResult = null;

        // Theme management
        function initTheme() {
            const savedTheme = localStorage.getItem('theme') || 'dark';
            if (savedTheme === 'light') {
                document.body.classList.add('light-mode');
                document.getElementById('themeIcon').textContent = '☀️';
                document.getElementById('themeText').textContent = 'Light';
            }
        }

        function toggleTheme() {
            document.body.classList.toggle('light-mode');
            const isLight = document.body.classList.contains('light-mode');
            document.getElementById('themeIcon').textContent = isLight ? '☀️' : '🌙';
            document.getElementById('themeText').textContent = isLight ? 'Light' : 'Dark';
            localStorage.setItem('theme', isLight ? 'light' : 'dark');
        }

        // Initialize theme on load
        initTheme();

        // Tab switching
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', () => {
                document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
                tab.classList.add('active');
                currentEncoding = tab.dataset.encoding;

                // Update placeholder
                const placeholder = currentEncoding === 'hex'
                    ? 'Paste your hex encoded blob here (with or without 0x prefix)...'
                    : 'Paste your base64 encoded blob here...';
                document.getElementById('blobInput').placeholder = placeholder;
            });
        });

        // View tab switching
        document.querySelectorAll('.view-tab').forEach(tab => {
            tab.addEventListener('click', () => {
                document.querySelectorAll('.view-tab').forEach(t => t.classList.remove('active'));
                tab.classList.add('active');

                document.querySelectorAll('.view-content').forEach(v => v.classList.remove('active'));
                document.getElementById(tab.dataset.view + 'View').classList.add('active');
            });
        });

        async function decodeBlob() {
            const input = document.getElementById('blobInput').value.trim();
            if (!input) {
                showError('Please enter some blob data');
                return;
            }

            // Hide previous results/errors
            document.getElementById('error').style.display = 'none';
            document.getElementById('result').classList.remove('active');
            document.getElementById('loading').style.display = 'block';

            try {
                const response = await fetch('/api/decode', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        data: input,
                        encoding: currentEncoding
                    })
                });

                const result = await response.json();
                currentResult = result;

                if (result.error) {
                    showError(result.error);
                } else {
                    displayResult(result);
                }
            } catch (err) {
                showError('Failed to decode blob: ' + err.message);
            } finally {
                document.getElementById('loading').style.display = 'none';
            }
        }

        function showError(message) {
            const errorDiv = document.getElementById('error');
            errorDiv.textContent = message;
            errorDiv.style.display = 'block';
        }

        function displayResult(result) {
            // Update header
            document.getElementById('blobType').textContent = 'Type: ' + result.type;
            document.getElementById('blobSize').textContent = 'Size: ' + formatBytes(result.size);

            // Parsed view
            document.getElementById('parsedView').innerHTML = formatParsedData(result);

            // Raw JSON view
            document.getElementById('rawView').innerHTML =
                '<pre>' + JSON.stringify(result.data, null, 2) + '</pre>';

            // Hex dump view
            document.getElementById('hexView').innerHTML =
                '<pre>' + formatHexDump(result.rawHex) + '</pre>';

            // Show result
            document.getElementById('result').classList.add('active');
        }

        function formatParsedData(result) {
            let html = '';

            if (result.type === 'SignedHeader') {
                const data = result.data;
                html = ` + "`" + `
                    <div class="field">
                        <span class="field-name">Chain ID:</span>
                        <span class="field-value">${data.chainId || 'N/A'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Height:</span>
                        <span class="field-value">${data.height}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Time:</span>
                        <span class="field-value">${data.time}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Version:</span>
                        <span class="field-value">Block: ${data.version.block}, App: ${data.version.app}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Last Header Hash:</span>
                        <span class="field-value hash">${data.lastHeaderHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Last Commit Hash:</span>
                        <span class="field-value hash">${data.lastCommitHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Version:</span>
                        <span class="field-value">Block: ${data.version.block}, App: ${data.version.app}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Last Header Hash:</span>
                        <span class="field-value">${data.lastHeaderHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Last Commit Hash:</span>
                        <span class="field-value">${data.lastCommitHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Data Hash:</span>
                        <span class="field-value hash">${data.dataHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Consensus Hash:</span>
                        <span class="field-value">${data.consensusHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">App Hash:</span>
                        <span class="field-value">${data.appHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Last Results Hash:</span>
                        <span class="field-value">${data.lastResultsHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Validator Hash:</span>
                        <span class="field-value">${data.validatorHash || 'empty'}</span>
                        <span class="field-name">Consensus Hash:</span>
                        <span class="field-value hash">${data.consensusHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">App Hash:</span>
                        <span class="field-value hash">${data.appHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Last Results Hash:</span>
                        <span class="field-value hash">${data.lastResultsHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Validator Hash:</span>
                        <span class="field-value hash">${data.validatorHash || 'empty'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Proposer Address:</span>
                        <span class="field-value hash">${data.proposerAddress || 'N/A'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Signature:</span>
                        <span class="field-value hash">${data.signature || 'N/A'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Signer Address:</span>
                        <span class="field-value hash">${data.signer?.address || 'N/A'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Signer Address:</span>
                        <span class="field-value">${data.signer.address || 'N/A'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Signer PubKey:</span>
                        <span class="field-value">${data.signer.pubKey || 'N/A'}</span>
                    </div>
                ` + "`" + `;
            } else if (result.type === 'SignedData') {
                const data = result.data;
                html = ` + "`" + `
                    ${data.chainId ? ` + "`" + `<div class="field">
                        <span class="field-name">Chain ID:</span>
                        <span class="field-value">${data.chainId}</span>
                    </div>` + "`" + ` : ''}
                    ${data.height !== undefined ? ` + "`" + `<div class="field">
                        <span class="field-name">Height:</span>
                        <span class="field-value">${data.height}</span>
                    </div>` + "`" + ` : ''}
                    ${data.time ? ` + "`" + `<div class="field">
                        <span class="field-name">Time:</span>
                        <span class="field-value">${data.time}</span>
                    </div>` + "`" + ` : ''}
                    ${data.lastDataHash ? ` + "`" + `<div class="field">
                        <span class="field-name">Last Data Hash:</span>
                        <span class="field-value hash">${data.lastDataHash}</span>
                    </div>` + "`" + ` : ''}
                    <div class="field">
                        <span class="field-name">Data Hash (DACommitment):</span>
                        <span class="field-value hash">${data.dataHash}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Transaction Count:</span>
                        <span class="field-value">${data.transactionCount}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Signer Address:</span>
                        <span class="field-value hash">${data.signer?.address || 'N/A'}</span>
                    </div>
                    <div class="field">
                        <span class="field-name">Signature:</span>
                        <span class="field-value hash">${data.signature}</span>
                    </div>
                ` + "`" + `;

                if (data.transactions && data.transactions.length > 0) {
                    html += '<div class="tx-list"><h3>Transactions:</h3>';
                    data.transactions.forEach(tx => {
                        html += ` + "`" + `
                            <div class="tx-item">
                                <div class="tx-header">
                                    <span>Transaction #${tx.index + 1}</span>
                                    <span>${formatBytes(tx.size)}</span>
                                </div>
                                <div class="tx-hash">Hash: ${tx.hash}</div>
                                <div class="tx-hash">Data (first 100 chars): ${tx.data.substring(0, 100)}...</div>
                            </div>
                        ` + "`" + `;
                    });
                    html += '</div>';
                }
            } else if (result.type === 'JSON') {
                html = '<pre>' + JSON.stringify(result.data, null, 2) + '</pre>';
            } else {
                html = '<pre>' + JSON.stringify(result.data, null, 2) + '</pre>';
            }

            return html;
        }

        function formatHexDump(hexString) {
            if (!hexString) return '';

            let result = '';
            const bytes = hexString.match(/.{1,2}/g) || [];

            for (let i = 0; i < bytes.length; i += 16) {
                // Offset
                result += i.toString(16).padStart(8, '0') + '  ';

                // Hex bytes
                for (let j = 0; j < 16; j++) {
                    if (i + j < bytes.length) {
                        result += bytes[i + j] + ' ';
                    } else {
                        result += '   ';
                    }
                    if (j === 7) result += ' ';
                }

                result += ' |';

                // ASCII
                for (let j = 0; j < 16 && i + j < bytes.length; j++) {
                    const byte = parseInt(bytes[i + j], 16);
                    result += (byte >= 32 && byte <= 126) ? String.fromCharCode(byte) : '.';
                }

                result += '|\n';
            }

            return result;
        }

        function formatBytes(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        // Allow Enter key to decode
        document.getElementById('blobInput').addEventListener('keydown', (e) => {
            if (e.ctrlKey && e.key === 'Enter') {
                decodeBlob();
            }
        });
    </script>
</body>
</html>
`

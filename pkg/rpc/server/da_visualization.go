package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/rs/zerolog"
)

// DASubmissionInfo represents information about a DA submission
type DASubmissionInfo struct {
	ID          string    `json:"id"`
	Height      uint64    `json:"height"`
	BlobSize    uint64    `json:"blob_size"`
	Timestamp   time.Time `json:"timestamp"`
	GasPrice    float64   `json:"gas_price"`
	StatusCode  string    `json:"status_code"`
	Message     string    `json:"message,omitempty"`
	NumBlobs    int       `json:"num_blobs"`
	BlobIDs     []string  `json:"blob_ids,omitempty"`
}

// DAVisualizationServer provides DA layer visualization endpoints
type DAVisualizationServer struct {
	da         coreda.DA
	logger     zerolog.Logger
	submissions []DASubmissionInfo
	mutex      sync.RWMutex
}

// NewDAVisualizationServer creates a new DA visualization server
func NewDAVisualizationServer(da coreda.DA, logger zerolog.Logger) *DAVisualizationServer {
	return &DAVisualizationServer{
		da:          da,
		logger:      logger,
		submissions: make([]DASubmissionInfo, 0),
	}
}

// RecordSubmission records a DA submission for visualization
func (s *DAVisualizationServer) RecordSubmission(result *coreda.ResultSubmit, gasPrice float64, numBlobs int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	statusCode := s.getStatusCodeString(result.Code)
	blobIDs := make([]string, len(result.IDs))
	for i, id := range result.IDs {
		blobIDs[i] = hex.EncodeToString(id)
	}

	submission := DASubmissionInfo{
		ID:          fmt.Sprintf("submission_%d_%d", result.Height, time.Now().Unix()),
		Height:      result.Height,
		BlobSize:    result.BlobSize,
		Timestamp:   result.Timestamp,
		GasPrice:    gasPrice,
		StatusCode:  statusCode,
		Message:     result.Message,
		NumBlobs:    numBlobs,
		BlobIDs:     blobIDs,
	}

	// Keep only the last 100 submissions to avoid memory growth
	s.submissions = append(s.submissions, submission)
	if len(s.submissions) > 100 {
		s.submissions = s.submissions[1:]
	}
}

// getStatusCodeString converts status code to human-readable string
func (s *DAVisualizationServer) getStatusCodeString(code coreda.StatusCode) string {
	switch code {
	case coreda.StatusSuccess:
		return "Success"
	case coreda.StatusNotFound:
		return "Not Found"
	case coreda.StatusNotIncludedInBlock:
		return "Not Included In Block"
	case coreda.StatusAlreadyInMempool:
		return "Already In Mempool"
	case coreda.StatusTooBig:
		return "Too Big"
	case coreda.StatusContextDeadline:
		return "Context Deadline"
	case coreda.StatusError:
		return "Error"
	case coreda.StatusIncorrectAccountSequence:
		return "Incorrect Account Sequence"
	case coreda.StatusContextCanceled:
		return "Context Canceled"
	case coreda.StatusHeightFromFuture:
		return "Height From Future"
	default:
		return "Unknown"
	}
}

// handleDASubmissions returns JSON list of recent DA submissions
func (s *DAVisualizationServer) handleDASubmissions(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"submissions": s.submissions,
		"total":       len(s.submissions),
	}); err != nil {
		s.logger.Error().Err(err).Msg("Failed to encode DA submissions response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleDABlobDetails returns details about a specific blob
func (s *DAVisualizationServer) handleDABlobDetails(w http.ResponseWriter, r *http.Request) {
	blobID := r.URL.Query().Get("id")
	if blobID == "" {
		http.Error(w, "Missing blob ID parameter", http.StatusBadRequest)
		return
	}

	// Decode the hex blob ID
	id, err := hex.DecodeString(blobID)
	if err != nil {
		http.Error(w, "Invalid blob ID format", http.StatusBadRequest)
		return
	}

	// Try to retrieve blob from DA layer
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Extract namespace - using empty namespace for now, could be parameterized
	namespace := []byte{}
	blobs, err := s.da.Get(ctx, []coreda.ID{id}, namespace)
	if err != nil {
		s.logger.Error().Err(err).Str("blob_id", blobID).Msg("Failed to retrieve blob from DA")
		http.Error(w, fmt.Sprintf("Failed to retrieve blob: %v", err), http.StatusInternalServerError)
		return
	}

	if len(blobs) == 0 {
		http.Error(w, "Blob not found", http.StatusNotFound)
		return
	}

	// Parse the blob ID to extract height and commitment
	height, commitment, err := coreda.SplitID(id)
	if err != nil {
		s.logger.Error().Err(err).Str("blob_id", blobID).Msg("Failed to split blob ID")
	}

	blob := blobs[0]
	response := map[string]interface{}{
		"id":          blobID,
		"height":      height,
		"commitment":  hex.EncodeToString(commitment),
		"size":        len(blob),
		"content":     hex.EncodeToString(blob),
		"content_preview": string(blob[:min(len(blob), 200)]), // First 200 bytes as string
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error().Err(err).Msg("Failed to encode blob details response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleDAVisualizationHTML returns HTML visualization page
func (s *DAVisualizationServer) handleDAVisualizationHTML(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	submissions := make([]DASubmissionInfo, len(s.submissions))
	copy(submissions, s.submissions)
	s.mutex.RUnlock()

	// Reverse the slice to show newest first
	for i, j := 0, len(submissions)-1; i < j; i, j = i+1, j-1 {
		submissions[i], submissions[j] = submissions[j], submissions[i]
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>DA Layer Visualization</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f5f5f5; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .submission { border: 1px solid #ddd; margin: 10px 0; padding: 15px; border-radius: 5px; }
        .success { border-left: 4px solid #4CAF50; }
        .error { border-left: 4px solid #f44336; }
        .pending { border-left: 4px solid #ff9800; }
        .blob-ids { margin-top: 10px; }
        .blob-id { background-color: #f0f0f0; padding: 2px 6px; margin: 2px; border-radius: 3px; font-family: monospace; font-size: 12px; }
        .meta { color: #666; font-size: 14px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .blob-link { color: #007cba; text-decoration: none; }
        .blob-link:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="header">
        <h1>DA Layer Visualization</h1>
        <p>Real-time view of blob submissions from the sequencer node to the Data Availability layer.</p>
        <p><strong>Total Submissions:</strong> {{len .}} | <strong>Last Update:</strong> {{.LastUpdate}}</p>
    </div>

    <h2>Recent Submissions</h2>
    {{if .}}
    <table>
        <tr>
            <th>Timestamp</th>
            <th>Height</th>
            <th>Status</th>
            <th>Blobs</th>
            <th>Size (bytes)</th>
            <th>Gas Price</th>
            <th>Message</th>
        </tr>
        {{range .}}
        <tr class="{{if eq .StatusCode "Success"}}success{{else if eq .StatusCode "Error"}}error{{else}}pending{{end}}">
            <td>{{.Timestamp.Format "15:04:05"}}</td>
            <td>{{.Height}}</td>
            <td>{{.StatusCode}}</td>
            <td>
                {{.NumBlobs}}
                {{if .BlobIDs}}
                <div class="blob-ids">
                    {{range .BlobIDs}}
                    <a href="/da/blob?id={{.}}" class="blob-link blob-id">{{slice . 0 8}}...</a>
                    {{end}}
                </div>
                {{end}}
            </td>
            <td>{{.BlobSize}}</td>
            <td>{{printf "%.6f" .GasPrice}}</td>
            <td>{{.Message}}</td>
        </tr>
        {{end}}
    </table>
    {{else}}
    <p>No submissions recorded yet.</p>
    {{end}}

    <div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; color: #666;">
        <p><strong>API Endpoints:</strong></p>
        <ul>
            <li><code>GET /da/submissions</code> - JSON list of submissions</li>
            <li><code>GET /da/blob?id=&lt;blob_id&gt;</code> - Detailed blob information</li>
        </ul>
        <p><em>Auto-refresh: <span id="countdown">30</span>s</em></p>
    </div>

    <script>
        // Auto-refresh page every 30 seconds
        let countdown = 30;
        const countdownEl = document.getElementById('countdown');
        setInterval(() => {
            countdown--;
            countdownEl.textContent = countdown;
            if (countdown <= 0) {
                location.reload();
            }
        }, 1000);
    </script>
</body>
</html>
`

	t, err := template.New("da").Funcs(template.FuncMap{
		"slice": func(s string, start, end int) string {
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"len": func(s []DASubmissionInfo) int {
			return len(s)
		},
	}).Parse(tmpl)

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to parse template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create template data with LastUpdate
	data := struct {
		Submissions []DASubmissionInfo
		LastUpdate  string
	}{
		Submissions: submissions,
		LastUpdate:  time.Now().Format("15:04:05"),
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, data.Submissions); err != nil {
		s.logger.Error().Err(err).Msg("Failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Global DA visualization server instance
var daVisualizationServer *DAVisualizationServer
var daVisualizationMutex sync.Mutex

// SetDAVisualizationServer sets the global DA visualization server instance
func SetDAVisualizationServer(server *DAVisualizationServer) {
	daVisualizationMutex.Lock()
	defer daVisualizationMutex.Unlock()
	daVisualizationServer = server
}

// GetDAVisualizationServer returns the global DA visualization server instance
func GetDAVisualizationServer() *DAVisualizationServer {
	daVisualizationMutex.Lock()
	defer daVisualizationMutex.Unlock()
	return daVisualizationServer
}
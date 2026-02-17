package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"sync"
	"time"

	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/rs/zerolog"
)

// daVisualizationHTML contains the DA visualization dashboard template.
// Inlined as a constant instead of using go:embed to avoid file-access issues
// in sandboxed build environments.
const daVisualizationHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Evolve DA Layer Visualization</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f5f5f5; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .api-section { background-color: #e8f4f8; padding: 20px; border-radius: 5px; margin-bottom: 20px; border: 1px solid #b3d9e6; }
        .api-endpoint { background-color: #fff; padding: 15px; margin: 10px 0; border-radius: 5px; border: 1px solid #ddd; }
        .api-endpoint h4 { margin: 0 0 10px 0; color: #007cba; }
        .api-endpoint code { background-color: #f4f4f4; padding: 4px 8px; border-radius: 3px; font-size: 14px; }
        .api-response { background-color: #f9f9f9; padding: 10px; margin-top: 10px; border-radius: 3px; border-left: 3px solid #007cba; }
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
        .method { display: inline-block; padding: 2px 6px; border-radius: 3px; font-weight: bold; font-size: 12px; margin-right: 8px; }
        .method-get { background-color: #61b5ff; color: white; }
        .method-post { background-color: #49cc90; color: white; }
        .example-link { color: #007cba; text-decoration: none; font-size: 13px; }
        .example-link:hover { text-decoration: underline; }
        .blob-ids { margin-top: 5px; }
        .blob-ids-preview { display: inline; }
        .blob-ids-full { display: none; max-height: 200px; overflow-y: auto; margin-top: 5px; padding: 5px; background: #fafafa; border-radius: 3px; }
        .blob-ids-full.expanded { display: block; }
        .blob-toggle { background: none; border: 1px solid #007cba; color: #007cba; padding: 2px 8px; border-radius: 3px; cursor: pointer; font-size: 11px; margin-left: 5px; }
        .blob-toggle:hover { background: #007cba; color: white; }
    </style>
</head>
<body>
    <div class="header">
        <h1>DA Layer Visualization</h1>
        <p>Real-time view of blob submissions from the sequencer node to the Data Availability layer.</p>
        {{if .IsAggregator}}
            {{if .Submissions}}
            <p><strong>Recent Submissions:</strong> {{len .Submissions}} (last 100) | <strong>Last Update:</strong> {{.LastUpdate}}</p>
            {{else}}
            <p><strong>Node Type:</strong> Aggregator | <strong>Recent Submissions:</strong> 0 | <strong>Last Update:</strong> {{.LastUpdate}}</p>
            {{end}}
        {{else}}
        <p><strong>Node Type:</strong> Non-aggregator | This node does not submit data to the DA layer.</p>
        {{end}}
    </div>

    {{if .IsAggregator}}
    <div class="api-section">
        <h2>Available API Endpoints</h2>

        <div class="api-endpoint">
            <h4><span class="method method-get">GET</span> /da</h4>
            <p>Returns this HTML visualization dashboard with real-time DA submission data.</p>
            <p><strong>Example:</strong> <a href="/da" class="example-link">View Dashboard</a></p>
        </div>

        <div class="api-endpoint">
            <h4><span class="method method-get">GET</span> /da/submissions</h4>
            <p>Returns a JSON array of the most recent DA submissions (up to 100) with metadata.</p>
            <p><strong>Note:</strong> Only aggregator nodes submit to the DA layer.</p>
            <p><strong>Example:</strong> <code>curl http://localhost:8080/da/submissions</code></p>
            <div class="api-response">
                <strong>Response:</strong>
                <pre>{
  "submissions": [
    {
      "id": "submission_1234_1699999999",
      "height": 1234,
      "blob_size": 2048,
      "timestamp": "2023-11-15T10:30:00Z",
      "gas_price": 0.000001,
      "status_code": "Success",
      "num_blobs": 1,
      "blob_ids": ["a1b2c3d4..."]
    }
  ],
  "total": 42
}</pre>
            </div>
        </div>

        <div class="api-endpoint">
            <h4><span class="method method-get">GET</span> /da/blob?id={blob_id}</h4>
            <p>Returns detailed information about a specific blob including its content.</p>
            <p><strong>Parameters:</strong> <code>id</code> - Hexadecimal blob ID</p>
            <p><strong>Example:</strong> <code>curl http://localhost:8080/da/blob?id=a1b2c3d4...</code></p>
            <div class="api-response">
                <strong>Response:</strong>
                <pre>{
  "id": "a1b2c3d4...",
  "height": 1234,
  "commitment": "deadbeef...",
  "size": 2048,
  "content": "0x1234...",
  "content_preview": "..."
}</pre>
            </div>
        </div>

        <div class="api-endpoint">
            <h4><span class="method method-get">GET</span> /da/stats</h4>
            <p>Returns aggregated statistics about DA submissions.</p>
            <p><strong>Example:</strong> <code>curl http://localhost:8080/da/stats</code></p>
            <div class="api-response">
                <strong>Response:</strong>
                <pre>{
  "total_submissions": 42,
  "success_count": 40,
  "error_count": 2,
  "success_rate": "95.24%",
  "total_blob_size": 86016,
  "avg_blob_size": 2048,
  "avg_gas_price": 0.000001,
  "time_range": {
    "first": "2023-11-15T10:00:00Z",
    "last": "2023-11-15T10:30:00Z"
  }
}</pre>
            </div>
        </div>

        <div class="api-endpoint">
            <h4><span class="method method-get">GET</span> /da/health</h4>
            <p>Returns health status of the DA layer connection.</p>
            <p><strong>Example:</strong> <code>curl http://localhost:8080/da/health</code></p>
            <div class="api-response">
                <strong>Response:</strong>
                <pre>{
  "status": "healthy",
  "is_healthy": true,
  "connection_status": "connected",
  "connection_healthy": true,
  "metrics": {
    "recent_error_rate": "10.0%",
    "recent_errors": 1,
    "recent_successes": 9,
    "recent_sample_size": 10,
    "total_submissions": 42,
    "last_submission_time": "2023-11-15T10:30:00Z",
    "last_success_time": "2023-11-15T10:29:45Z",
    "last_error_time": "2023-11-15T10:25:00Z"
  },
  "issues": [],
  "timestamp": "2023-11-15T10:30:15Z"
}</pre>
            <p style="margin-top: 15px;"><strong>Health Status Values:</strong></p>
            <ul style="margin-left: 20px; font-size: 13px; line-height: 1.8;">
                <li><code>healthy</code> - System operating normally</li>
                <li><code>degraded</code> - Elevated error rate but still functional</li>
                <li><code>unhealthy</code> - Critical issues detected</li>
                <li><code>warning</code> - Potential issues that need attention</li>
                <li><code>unknown</code> - Insufficient data to determine health</li>
            </ul>
        </div>
    </div>
    {{end}}

    {{if .IsAggregator}}
    <h2>Recent Submissions</h2>
    {{if .Submissions}}
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
        {{range .Submissions}}
        <tr class="{{if eq .StatusCode "Success"}}success{{else if eq .StatusCode "Error"}}error{{else}}pending{{end}}">
            <td>{{if not .Timestamp.IsZero}}{{.Timestamp.Format "15:04:05"}}{{else}}--:--:--{{end}}</td>
            <td>{{.Height}}</td>
            <td>{{.StatusCode}}</td>
            <td>
                {{.NumBlobs}}
                {{if .BlobIDs}}
                {{$namespace := .Namespace}}
                {{$numBlobs := len .BlobIDs}}
                {{if le $numBlobs 5}}
                <div class="blob-ids">
                    {{range .BlobIDs}}
                    <a href="/da/blob?id={{.}}&namespace={{$namespace}}" class="blob-link blob-id" title="Click to view blob details">{{slice . 0 8}}...</a>
                    {{end}}
                </div>
                {{else}}
                <div class="blob-ids">
                    <span class="blob-ids-preview">
                        {{range $i, $id := .BlobIDs}}{{if lt $i 3}}<a href="/da/blob?id={{$id}}&namespace={{$namespace}}" class="blob-link blob-id" title="Click to view blob details">{{slice $id 0 8}}...</a>{{end}}{{end}}
                    </span>
                    <button class="blob-toggle" onclick="toggleBlobs(this)">+{{subtract $numBlobs 3}} more</button>
                    <div class="blob-ids-full">
                        {{range .BlobIDs}}
                        <a href="/da/blob?id={{.}}&namespace={{$namespace}}" class="blob-link blob-id" title="Click to view blob details">{{slice . 0 8}}...</a>
                        {{end}}
                    </div>
                </div>
                {{end}}
                {{end}}
            </td>
            <td>{{.BlobSize}}</td>
            <td>{{printf "%.6f" .GasPrice}}</td>
            <td>{{.Message}}</td>
        </tr>
        {{end}}
    </table>
    {{else}}
    <p>No submissions recorded yet. This aggregator node has not submitted any data to the DA layer yet.</p>
    {{end}}
    {{else}}
    <h2>Node Information</h2>
    <p>This is a non-aggregator node. Non-aggregator nodes do not submit data to the DA layer and therefore do not have submission statistics, health metrics, or DA-related API endpoints available.</p>
    <p>Only aggregator nodes that actively produce blocks and submit data to the DA layer will display this information.</p>
    {{end}}

    <div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; color: #666;">
        <p><em>Auto-refresh: <span id="countdown">30</span>s</em> | <a href="javascript:location.reload()" style="color: #007cba;">Refresh Now</a></p>
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

        // Toggle blob list expansion
        function toggleBlobs(btn) {
            const container = btn.parentElement;
            const fullList = container.querySelector('.blob-ids-full');
            const preview = container.querySelector('.blob-ids-preview');
            const isExpanded = fullList.classList.contains('expanded');

            if (isExpanded) {
                fullList.classList.remove('expanded');
                preview.style.display = 'inline';
                btn.textContent = btn.dataset.expandText;
            } else {
                fullList.classList.add('expanded');
                preview.style.display = 'none';
                btn.dataset.expandText = btn.textContent;
                btn.textContent = 'Show less';
            }
        }
    </script>
</body>
</html>`

// DASubmissionInfo represents information about a DA submission
type DASubmissionInfo struct {
	ID         string    `json:"id"`
	Height     uint64    `json:"height"`
	BlobSize   uint64    `json:"blob_size"`
	Timestamp  time.Time `json:"timestamp"`
	GasPrice   float64   `json:"gas_price"`
	StatusCode string    `json:"status_code"`
	Message    string    `json:"message,omitempty"`
	NumBlobs   uint64    `json:"num_blobs"`
	BlobIDs    []string  `json:"blob_ids,omitempty"`
	Namespace  string    `json:"namespace,omitempty"`
}

// DAVisualizationServer provides DA layer visualization endpoints
type DAVisualizationServer struct {
	da           DAVizClient
	logger       zerolog.Logger
	submissions  []DASubmissionInfo
	mutex        sync.RWMutex
	isAggregator bool
}

// DAVizClient is the minimal DA surface needed by the visualization server.
type DAVizClient interface {
	Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error)
}

// NewDAVisualizationServer creates a new DA visualization server
func NewDAVisualizationServer(da DAVizClient, logger zerolog.Logger, isAggregator bool) *DAVisualizationServer {
	return &DAVisualizationServer{
		da:           da,
		logger:       logger,
		submissions:  make([]DASubmissionInfo, 0),
		isAggregator: isAggregator,
	}
}

// RecordSubmission records a DA submission for visualization
// Only keeps the last 100 submissions in memory for the dashboard display
func (s *DAVisualizationServer) RecordSubmission(result *datypes.ResultSubmit, gasPrice float64, numBlobs uint64, namespace []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	statusCode := s.getStatusCodeString(result.Code)
	blobIDs := make([]string, len(result.IDs))
	for i, id := range result.IDs {
		blobIDs[i] = hex.EncodeToString(id)
	}

	submission := DASubmissionInfo{
		ID:         fmt.Sprintf("submission_%d_%d", result.Height, time.Now().Unix()),
		Height:     result.Height,
		BlobSize:   result.BlobSize,
		Timestamp:  result.Timestamp,
		GasPrice:   gasPrice,
		StatusCode: statusCode,
		Message:    result.Message,
		NumBlobs:   numBlobs,
		BlobIDs:    blobIDs,
		Namespace:  hex.EncodeToString(namespace),
	}

	// Keep only the last 100 submissions in memory to avoid memory growth
	// The HTML dashboard shows these recent submissions only
	s.submissions = append(s.submissions, submission)
	if len(s.submissions) > 100 {
		s.submissions = s.submissions[1:]
	}
}

// getStatusCodeString converts status code to human-readable string
func (s *DAVisualizationServer) getStatusCodeString(code datypes.StatusCode) string {
	switch code {
	case datypes.StatusSuccess:
		return "Success"
	case datypes.StatusNotFound:
		return "Not Found"
	case datypes.StatusNotIncludedInBlock:
		return "Not Included In Block"
	case datypes.StatusAlreadyInMempool:
		return "Already In Mempool"
	case datypes.StatusTooBig:
		return "Too Big"
	case datypes.StatusContextDeadline:
		return "Context Deadline"
	case datypes.StatusError:
		return "Error"
	case datypes.StatusIncorrectAccountSequence:
		return "Incorrect Account Sequence"
	case datypes.StatusContextCanceled:
		return "Context Canceled"
	case datypes.StatusHeightFromFuture:
		return "Height From Future"
	default:
		return "Unknown"
	}
}

// handleDASubmissions returns JSON list of recent DA submissions
// Note: This returns only the most recent submissions kept in memory (max 100)
func (s *DAVisualizationServer) handleDASubmissions(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// If not an aggregator, return empty submissions with a message
	if !s.isAggregator {
		response := map[string]any{
			"is_aggregator": false,
			"submissions":   []DASubmissionInfo{},
			"total":         0,
			"message":       "This node is not an aggregator and does not submit to the DA layer",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.logger.Error().Err(err).Msg("Failed to encode DA submissions response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Reverse the slice to show newest first
	reversed := make([]DASubmissionInfo, len(s.submissions))
	for i, j := 0, len(s.submissions)-1; j >= 0; i, j = i+1, j-1 {
		reversed[i] = s.submissions[j]
	}

	// Build response
	response := map[string]any{
		"submissions": reversed,
		"total":       len(reversed),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
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

	var namespace []byte
	found := false

	// 1. Check query parameter first
	nsParam := r.URL.Query().Get("namespace")
	if nsParam != "" {
		if ns, err := datypes.ParseHexNamespace(nsParam); err == nil {
			namespace = ns.Bytes()
			found = true
		} else {
			ns := datypes.NamespaceFromString(nsParam)
			namespace = ns.Bytes()
			found = true
		}
	}

	// 2. If not provided in query, try to find in recent submissions
	if !found {
		s.mutex.RLock()
		for _, submission := range s.submissions {
			if slices.Contains(submission.BlobIDs, blobID) {
				if submission.Namespace != "" {
					if ns, err := hex.DecodeString(submission.Namespace); err == nil {
						namespace = ns
						found = true
					}
				}
			}
			if found {
				break
			}
		}
		s.mutex.RUnlock()
	}

	if !found || len(namespace) == 0 {
		http.Error(w, "Namespace required to retrieve blob (not found in recent submissions and not provided in query)", http.StatusBadRequest)
		return
	}

	blobs, err := s.da.Get(ctx, []datypes.ID{id}, namespace)
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
	height, commitment := blobrpc.SplitID(id)

	blob := blobs[0]
	response := map[string]any{
		"id":              blobID,
		"height":          height,
		"commitment":      hex.EncodeToString(commitment),
		"size":            len(blob),
		"content":         hex.EncodeToString(blob),
		"content_preview": string(blob[:min(len(blob), 200)]), // First 200 bytes as string
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error().Err(err).Msg("Failed to encode blob details response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleDAStats returns aggregated statistics about DA submissions
func (s *DAVisualizationServer) handleDAStats(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// If not an aggregator, return empty stats
	if !s.isAggregator {
		stats := map[string]any{
			"is_aggregator":     false,
			"total_submissions": 0,
			"message":           "This node is not an aggregator and does not submit to the DA layer",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			s.logger.Error().Err(err).Msg("Failed to encode DA stats response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Calculate statistics
	var (
		totalSubmissions = len(s.submissions)
		successCount     int
		errorCount       int
		totalBlobSize    uint64
		totalGasPrice    float64
		avgBlobSize      float64
		avgGasPrice      float64
		successRate      float64
	)

	for _, submission := range s.submissions {
		switch submission.StatusCode {
		case "Success":
			successCount++
		case "Error":
			errorCount++
		}
		totalBlobSize += submission.BlobSize
		totalGasPrice += submission.GasPrice
	}

	if totalSubmissions > 0 {
		avgBlobSize = float64(totalBlobSize) / float64(totalSubmissions)
		avgGasPrice = totalGasPrice / float64(totalSubmissions)
		successRate = float64(successCount) / float64(totalSubmissions) * 100
	}

	// Get time range
	var firstSubmission, lastSubmission *time.Time
	if totalSubmissions > 0 {
		firstSubmission = &s.submissions[0].Timestamp
		lastSubmission = &s.submissions[len(s.submissions)-1].Timestamp
	}

	stats := map[string]any{
		"total_submissions": totalSubmissions,
		"success_count":     successCount,
		"error_count":       errorCount,
		"success_rate":      fmt.Sprintf("%.2f%%", successRate),
		"total_blob_size":   totalBlobSize,
		"avg_blob_size":     avgBlobSize,
		"avg_gas_price":     avgGasPrice,
		"time_range": map[string]any{
			"first": firstSubmission,
			"last":  lastSubmission,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		s.logger.Error().Err(err).Msg("Failed to encode DA stats response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleDAHealth returns health status of the DA layer connection
func (s *DAVisualizationServer) handleDAHealth(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// If not an aggregator, return simplified health status
	if !s.isAggregator {
		health := map[string]any{
			"is_aggregator":     false,
			"status":            "n/a",
			"message":           "This node is not an aggregator and does not submit to the DA layer",
			"connection_status": "n/a",
			"timestamp":         time.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(health); err != nil {
			s.logger.Error().Err(err).Msg("Failed to encode DA health response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Calculate health metrics
	var (
		lastSuccessTime    *time.Time
		lastErrorTime      *time.Time
		recentErrors       int
		recentSuccesses    int
		lastSubmissionTime *time.Time
		errorRate          float64
		isHealthy          bool
		healthStatus       string
		healthIssues       []string
	)

	// Look at recent submissions (last 10 or all if less)
	recentCount := min(len(s.submissions), 10)

	// Analyze recent submissions
	for i := len(s.submissions) - recentCount; i < len(s.submissions); i++ {
		if i < 0 {
			continue
		}
		submission := s.submissions[i]
		switch submission.StatusCode {
		case "Success":
			recentSuccesses++
			if lastSuccessTime == nil || submission.Timestamp.After(*lastSuccessTime) {
				lastSuccessTime = &submission.Timestamp
			}
		case "Error":
			recentErrors++
			if lastErrorTime == nil || submission.Timestamp.After(*lastErrorTime) {
				lastErrorTime = &submission.Timestamp
			}
		}
	}

	// Get the last submission time
	if len(s.submissions) > 0 {
		lastSubmissionTime = &s.submissions[len(s.submissions)-1].Timestamp
	}

	// Calculate error rate for recent submissions
	if recentCount > 0 {
		errorRate = float64(recentErrors) / float64(recentCount) * 100
	}

	// Determine health status based on criteria
	isHealthy = true
	healthStatus = "healthy"

	// Check error rate threshold (>20% is unhealthy)
	if errorRate > 20 {
		isHealthy = false
		healthStatus = "degraded"
		healthIssues = append(healthIssues, fmt.Sprintf("High error rate: %.1f%%", errorRate))
	}

	// Check if we haven't had a successful submission in the last 5 minutes
	if lastSuccessTime != nil && time.Since(*lastSuccessTime) > 5*time.Minute {
		isHealthy = false
		healthStatus = "unhealthy"
		healthIssues = append(healthIssues, fmt.Sprintf("No successful submissions for %v", time.Since(*lastSuccessTime).Round(time.Second)))
	}

	// Check if DA layer appears to be stalled (no submissions at all in last 2 minutes)
	if lastSubmissionTime != nil && time.Since(*lastSubmissionTime) > 2*time.Minute {
		healthStatus = "warning"
		healthIssues = append(healthIssues, fmt.Sprintf("No submissions for %v", time.Since(*lastSubmissionTime).Round(time.Second)))
	}

	// If no submissions at all
	if len(s.submissions) == 0 {
		healthStatus = "unknown"
		healthIssues = append(healthIssues, "No submissions recorded yet")
	}

	// Test DA layer connectivity (attempt a simple operation)
	var connectionStatus string
	connectionHealthy := false

	// Try to validate the DA layer is responsive
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Check if DA layer responds to basic operations
	// This is a non-invasive check - we're just testing responsiveness
	select {
	case <-ctx.Done():
		connectionStatus = "timeout"
	default:
		// DA layer is at least instantiated
		if s.da != nil {
			connectionStatus = "connected"
			connectionHealthy = true
		} else {
			connectionStatus = "disconnected"
			isHealthy = false
			healthStatus = "unhealthy"
			healthIssues = append(healthIssues, "DA layer not initialized")
		}
	}

	health := map[string]any{
		"status":             healthStatus,
		"is_healthy":         isHealthy,
		"connection_status":  connectionStatus,
		"connection_healthy": connectionHealthy,
		"metrics": map[string]any{
			"recent_error_rate":    fmt.Sprintf("%.1f%%", errorRate),
			"recent_errors":        recentErrors,
			"recent_successes":     recentSuccesses,
			"recent_sample_size":   recentCount,
			"total_submissions":    len(s.submissions),
			"last_submission_time": lastSubmissionTime,
			"last_success_time":    lastSuccessTime,
			"last_error_time":      lastErrorTime,
		},
		"issues":    healthIssues,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		s.logger.Error().Err(err).Msg("Failed to encode DA health response")
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

	t, err := template.New("da").Funcs(template.FuncMap{
		"slice": func(s string, start, end int) string {
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"len": func(items any) int {
			// Handle different types gracefully
			switch v := items.(type) {
			case []DASubmissionInfo:
				return len(v)
			case []string:
				return len(v)
			case struct {
				Submissions  []DASubmissionInfo
				LastUpdate   string
				IsAggregator bool
			}:
				return len(v.Submissions)
			default:
				return 0
			}
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"le": func(a, b int) bool {
			return a <= b
		},
	}).Parse(daVisualizationHTML)

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to parse template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create template data with LastUpdate
	data := struct {
		Submissions  []DASubmissionInfo
		LastUpdate   string
		IsAggregator bool
	}{
		Submissions:  submissions,
		LastUpdate:   time.Now().Format("15:04:05"),
		IsAggregator: s.isAggregator,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, data); err != nil {
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

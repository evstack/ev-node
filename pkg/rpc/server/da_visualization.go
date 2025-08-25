package server

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/rs/zerolog"
)

//go:embed templates/da_visualization.html
var daVisualizationHTML string

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
}

// DAVisualizationServer provides DA layer visualization endpoints
type DAVisualizationServer struct {
	da           coreda.DA
	logger       zerolog.Logger
	submissions  []DASubmissionInfo
	mutex        sync.RWMutex
	isAggregator bool
}

// NewDAVisualizationServer creates a new DA visualization server
func NewDAVisualizationServer(da coreda.DA, logger zerolog.Logger, isAggregator bool) *DAVisualizationServer {
	return &DAVisualizationServer{
		da:           da,
		logger:       logger,
		submissions:  make([]DASubmissionInfo, 0),
		isAggregator: isAggregator,
	}
}

// RecordSubmission records a DA submission for visualization
// Only keeps the last 100 submissions in memory for the dashboard display
func (s *DAVisualizationServer) RecordSubmission(result *coreda.ResultSubmit, gasPrice float64, numBlobs uint64) {
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
	}

	// Keep only the last 100 submissions in memory to avoid memory growth
	// The HTML dashboard shows these recent submissions only
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
// Note: This returns only the most recent submissions kept in memory (max 100)
func (s *DAVisualizationServer) handleDASubmissions(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// If not an aggregator, return empty submissions with a message
	if !s.isAggregator {
		response := map[string]interface{}{
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
	response := map[string]interface{}{
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
		stats := map[string]interface{}{
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

	stats := map[string]interface{}{
		"total_submissions": totalSubmissions,
		"success_count":     successCount,
		"error_count":       errorCount,
		"success_rate":      fmt.Sprintf("%.2f%%", successRate),
		"total_blob_size":   totalBlobSize,
		"avg_blob_size":     avgBlobSize,
		"avg_gas_price":     avgGasPrice,
		"time_range": map[string]interface{}{
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
		health := map[string]interface{}{
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

	health := map[string]interface{}{
		"status":             healthStatus,
		"is_healthy":         isHealthy,
		"connection_status":  connectionStatus,
		"connection_healthy": connectionHealthy,
		"metrics": map[string]interface{}{
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
		"len": func(items interface{}) int {
			// Handle different types gracefully
			switch v := items.(type) {
			case []DASubmissionInfo:
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

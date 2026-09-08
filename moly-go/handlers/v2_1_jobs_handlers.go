package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"moly/services"
)

// JobsHandlers manages job scheduler endpoints
type JobsHandlers struct {
	scheduler *services.JobScheduler
}

// NewJobsHandlers creates handlers for job management endpoints
func NewJobsHandlers(scheduler *services.JobScheduler) *JobsHandlers {
	if scheduler == nil {
		log.Fatal("[JobsHandlers] JobScheduler cannot be nil")
	}
	return &JobsHandlers{
		scheduler: scheduler,
	}
}

// RegisterRoutes registers all job management routes
func (jh *JobsHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2.1/jobs/status", jh.handleGetStatus)
	mux.HandleFunc("GET /api/v2.1/jobs/metrics", jh.handleGetMetrics)
	mux.HandleFunc("POST /api/v2.1/jobs/extraction/force", jh.handleForceExtraction)
	mux.HandleFunc("POST /api/v2.1/jobs/cleanup/force", jh.handleForceCleanup)
	mux.HandleFunc("POST /api/v2.1/jobs/metrics/reset", jh.handleResetMetrics)
}

// handleGetStatus GET /api/v2.1/jobs/status
func (jh *JobsHandlers) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	status := jh.scheduler.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":      status,
		"message":   "Job scheduler status",
		"timestamp": 0,
	})
}

// handleGetMetrics GET /api/v2.1/jobs/metrics
func (jh *JobsHandlers) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := jh.scheduler.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"extraction": map[string]interface{}{
				"jobs_run":          metrics.ExtractionJobsRun,
				"successful":        metrics.ExtractionSuccessful,
				"failed":            metrics.ExtractionFailed,
				"items_processed":   metrics.ExtractionItemsProcessed,
				"total_time_ms":     metrics.ExtractionTotalTimeMs,
			},
			"cleanup": map[string]interface{}{
				"jobs_run":                metrics.CleanupJobsRun,
				"successful":              metrics.CleanupSuccessful,
				"failed":                  metrics.CleanupFailed,
				"conversations_deleted":   metrics.CleanupConversationsDeleted,
				"total_time_ms":           metrics.CleanupTotalTimeMs,
			},
		},
		"message":   "Job scheduler metrics",
		"timestamp": 0,
	})
}

// handleForceExtraction POST /api/v2.1/jobs/extraction/force
func (jh *JobsHandlers) handleForceExtraction(w http.ResponseWriter, r *http.Request) {
	err := jh.scheduler.ForceExtractionNow()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Service Unavailable",
			"message": err.Error(),
			"code":    http.StatusServiceUnavailable,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":      map[string]interface{}{"triggered": true},
		"message":   "Extraction job triggered",
		"timestamp": 0,
	})

	log.Println("[JobsHandlers] Extraction job manually triggered")
}

// handleForceCleanup POST /api/v2.1/jobs/cleanup/force
func (jh *JobsHandlers) handleForceCleanup(w http.ResponseWriter, r *http.Request) {
	err := jh.scheduler.ForceCleanupNow()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Service Unavailable",
			"message": err.Error(),
			"code":    http.StatusServiceUnavailable,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":      map[string]interface{}{"triggered": true},
		"message":   "Cleanup job triggered",
		"timestamp": 0,
	})

	log.Println("[JobsHandlers] Cleanup job manually triggered")
}

// handleResetMetrics POST /api/v2.1/jobs/metrics/reset
func (jh *JobsHandlers) handleResetMetrics(w http.ResponseWriter, r *http.Request) {
	jh.scheduler.ResetMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":      map[string]interface{}{"reset": true},
		"message":   "Metrics reset",
		"timestamp": 0,
	})

	log.Println("[JobsHandlers] Metrics reset")
}

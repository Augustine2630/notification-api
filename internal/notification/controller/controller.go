package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"notification-api/internal/notification/service"
)

// Controller exposes the notification service over HTTP.
type Controller struct {
	notifier *service.NotificationService
}

func NewController(notifier *service.NotificationService) *Controller {
	return &Controller{notifier: notifier}
}

// RegisterRoutes wires the controller's HTTP handlers onto the given mux.
func (c *Controller) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/notifications/approve", c.HandleApprove)
	mux.HandleFunc("/api/v1/notifications/announce", c.HandleAnnounce)
}

// AnnounceRequest is the request body for POST /api/v1/notifications/announce.
type AnnounceRequest struct {
	Text      string  `json:"text"`
	ImageLink string  `json:"image_link"`
	UserIDs   []int64 `json:"user_ids"`
}

// AnnounceResponse is the response body for POST /api/v1/notifications/announce.
type AnnounceResponse struct {
	Status    string   `json:"status"`
	Message   string   `json:"message"`
	Sent      int      `json:"sent"`
	Failed    int      `json:"failed"`
	FailedIDs []int64  `json:"failed_ids,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

// HandleApprove sends an approval/rejection message to a user.
// GET/POST /api/v1/notifications/approve?telegramId=<id>&approved=true|false
func (c *Controller) HandleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	telegramIDStr := r.URL.Query().Get("telegramId")
	if telegramIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "telegramId is required")
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid telegramId")
		return
	}

	approvedStr := r.URL.Query().Get("approved")
	approved := approvedStr == "" || (approvedStr != "false" && approvedStr != "0")

	messageText, err := c.notifier.SendApprove(telegramID, approved)
	if err != nil {
		log.Println("Failed to send approve message:", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to send message")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"message":  messageText,
		"approved": approved,
	})
}

// HandleAnnounce fans a message out to a list of Telegram user IDs.
// POST /api/v1/notifications/announce
func (c *Controller) HandleAnnounce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req AnnounceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Text == "" {
		writeJSONError(w, http.StatusBadRequest, "text field is required")
		return
	}
	if len(req.UserIDs) == 0 {
		writeJSONError(w, http.StatusBadRequest, "user_ids array cannot be empty")
		return
	}

	result := c.notifier.SendAnnounce(req.UserIDs, req.Text, req.ImageLink)

	writeJSON(w, http.StatusOK, AnnounceResponse{
		Status:    "completed",
		Message:   "Sent to " + strconv.Itoa(result.Sent) + " users, " + strconv.Itoa(result.Failed) + " failed",
		Sent:      result.Sent,
		Failed:    result.Failed,
		FailedIDs: result.FailedIDs,
		Errors:    result.Errors,
	})
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

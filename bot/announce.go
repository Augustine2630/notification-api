package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// AnnounceRequest represents the request body for /api/v1/send/announce
type AnnounceRequest struct {
	Text      string  `json:"text"`
	ImageLink string  `json:"image_link"`
	UserIDs   []int64 `json:"user_ids"`
}

// AnnounceResponse represents the response body
type AnnounceResponse struct {
	Status    string   `json:"status"`
	Message   string   `json:"message"`
	Sent      int      `json:"sent"`
	Failed    int      `json:"failed"`
	FailedIDs []int64  `json:"failed_ids,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

// HandleAnnounce sends announcement messages to multiple users
func HandleAnnounce(ctx *HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req AnnounceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		// Validate required fields
		if req.Text == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "text field is required"})
			return
		}

		if len(req.UserIDs) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "user_ids array cannot be empty"})
			return
		}

		// Send announcements
		sent := 0
		failed := 0
		failedIDs := []int64{}
		errors := []string{}

		for _, userID := range req.UserIDs {
			// Send image if provided
			if req.ImageLink != "" {
				photo := tgbotapi.NewPhoto(userID, tgbotapi.FileURL(req.ImageLink))
				photo.Caption = req.Text
				photo.ParseMode = "HTML"
				if _, err := ctx.Bot.Send(photo); err != nil {
					log.Println("Failed to send image to user", userID, ":", err)
					failed++
					failedIDs = append(failedIDs, userID)
					errors = append(errors, fmt.Sprintf("User %d: %v", userID, err))
					continue
				}
			} else {
				// Send text message only
				msg := tgbotapi.NewMessage(userID, req.Text)
				msg.ParseMode = "HTML"
				if _, err := ctx.Bot.Send(msg); err != nil {
					log.Println("Failed to send message to user", userID, ":", err)
					failed++
					failedIDs = append(failedIDs, userID)
					errors = append(errors, fmt.Sprintf("User %d: %v", userID, err))
					continue
				}
			}
			sent++
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := AnnounceResponse{
			Status:    "completed",
			Message:   fmt.Sprintf("Sent to %d users, %d failed", sent, failed),
			Sent:      sent,
			Failed:    failed,
			FailedIDs: failedIDs,
			Errors:    errors,
		}
		json.NewEncoder(w).Encode(resp)
	}
}

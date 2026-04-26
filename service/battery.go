package service

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BatteryService struct {
	BotAPI           *tgbotapi.BotAPI
	NodeExporterHost string
	AlertChatIDs     []int64
	CheckInterval    time.Duration
	AlertThreshold   int
	done             chan struct{}
	lastAlertTime    time.Time
}

// NewBatteryService creates a new battery monitoring service
func NewBatteryService(botAPI *tgbotapi.BotAPI, nodeExporterHost string, alertChatIDs []int64) *BatteryService {
	return &BatteryService{
		BotAPI:           botAPI,
		NodeExporterHost: nodeExporterHost,
		AlertChatIDs:     alertChatIDs,
		CheckInterval:    120 * time.Second,
		AlertThreshold:   30,
		done:             make(chan struct{}),
		lastAlertTime:    time.Now().Add(-1 * time.Hour),
	}
}

// getBatteryCapacity fetches current battery capacity from node_exporter
func (bs *BatteryService) getBatteryCapacity() (int, error) {
	url := fmt.Sprintf("http://%s/metrics", bs.NodeExporterHost)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse battery capacity from metrics
	// Looking for: node_power_supply_current_capacity{power_supply="InternalBattery-0"} 100
	re := regexp.MustCompile(`node_power_supply_current_capacity\{power_supply="InternalBattery-0"\}\s+(\d+)`)
	matches := re.FindStringSubmatch(string(body))

	if len(matches) < 2 {
		return 0, fmt.Errorf("battery capacity metric not found")
	}

	capacity, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse battery capacity: %w", err)
	}

	return capacity, nil
}

// sendBatteryAlert sends alert message to configured chat IDs
func (bs *BatteryService) sendBatteryAlert(capacity int) {
	message := fmt.Sprintf("⚠️ <b>Низкий заряд батареи</b>\n\nТекущий заряд: <b>%d%%</b>\n\nПожалуйста, подключите зарядное устройство!", capacity)

	for _, chatID := range bs.AlertChatIDs {
		msg := tgbotapi.NewMessage(chatID, message)
		msg.ParseMode = "HTML"

		if _, err := bs.BotAPI.Send(msg); err != nil {
			log.Printf("Failed to send battery alert to chat %d: %v", chatID, err)
		} else {
			log.Printf("Battery alert sent to chat %d (capacity: %d%%)", chatID, capacity)
		}
	}
}

// Start begins the battery monitoring loop
func (bs *BatteryService) Start() {
	go func() {
		ticker := time.NewTicker(bs.CheckInterval)
		defer ticker.Stop()

		log.Printf("Battery monitoring started (check interval: %s, threshold: %d%%)", bs.CheckInterval, bs.AlertThreshold)

		for {
			select {
			case <-bs.done:
				log.Println("Battery monitoring stopped")
				return
			case <-ticker.C:
				capacity, err := bs.getBatteryCapacity()
				if err != nil {
					log.Printf("Failed to get battery capacity: %v", err)
					continue
				}

				log.Printf("Battery capacity: %d%%", capacity)

				// Send alert if capacity is below threshold and we haven't alerted in the last hour
				if capacity < bs.AlertThreshold && time.Since(bs.lastAlertTime) > 1*time.Hour {
					bs.sendBatteryAlert(capacity)
					bs.lastAlertTime = time.Now()
				}
			}
		}
	}()
}

// Stop gracefully stops the battery monitoring service
func (bs *BatteryService) Stop() {
	close(bs.done)
}

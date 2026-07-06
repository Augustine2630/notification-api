package vpnprofile

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"notification-api/internal/notification/service"
)

// BatteryService polls node_exporter for battery capacity and delegates the
// actual alert delivery to the notification service once the threshold is crossed.
type BatteryService struct {
	Notifier         *service.NotificationService
	NodeExporterHost string
	AlertChatIDs     []int64
	CheckInterval    time.Duration
	AlertThreshold   int
	done             chan struct{}
	lastAlertTime    time.Time
}

// NewBatteryService creates a new battery monitoring service
func NewBatteryService(notifier *service.NotificationService, nodeExporterHost string, alertChatIDs []int64) *BatteryService {
	return &BatteryService{
		Notifier:         notifier,
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

// sendBatteryAlert delegates alert delivery to the notification service.
func (bs *BatteryService) sendBatteryAlert(capacity int) {
	errs := bs.Notifier.SendBatteryAlert(bs.AlertChatIDs, capacity)
	for _, err := range errs {
		log.Printf("Failed to send battery alert: %v", err)
	}
	if len(errs) < len(bs.AlertChatIDs) {
		log.Printf("Battery alert sent (capacity: %d%%)", capacity)
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

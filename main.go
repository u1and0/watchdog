package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

type HealthMonitor struct {
	targetEndpoint  string
	slackWebhookURL string
	minInterval     time.Duration
	maxInterval     time.Duration
	logger          *log.Logger
	client          *http.Client
	errorCount      int
}

func NewHealthMonitor(endpoint, webhookURL string, minInterval, maxInterval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		targetEndpoint:  endpoint,
		slackWebhookURL: webhookURL,
		minInterval:     minInterval,
		maxInterval:     maxInterval,
		logger:          log.New(os.Stdout, "HealthMonitor: ", log.Ldate|log.Ltime|log.Lshortfile),
		client:          &http.Client{Timeout: 10 * time.Second},
		errorCount:      0,
	}
}

// calculateBackoffTime : 指数バックオフ（exponential backoff）アルゴリズムを使って
// チェック間隔を計算する
//
// minInterval := 1 * time.Second
// errorCount := 2
// backoffTime := minInterval * time.Duration(1<<uint(errorCount))
// backoffTime = 4秒
func calculateBackoffTime(errorCount int, minInterval, maxInterval time.Duration) time.Duration {
	backoffTime := minInterval * time.Duration(1<<uint(errorCount))
	if backoffTime > maxInterval {
		backoffTime = maxInterval
	}
	return backoffTime
}

func (h *HealthMonitor) sendSlackMessage(message string, isError bool) error {
	color := "good"
	if isError {
		color = "danger"
	}

	attachment := slack.Attachment{
		Color: color,
		Text:  message,
	}

	return slack.PostWebhook(h.slackWebhookURL, &slack.WebhookMessage{
		Text:        "Health Monitor Notification",
		Attachments: []slack.Attachment{attachment},
	})
}

func (h *HealthMonitor) checkHealth() {
	resp, err := h.client.Get(h.targetEndpoint)
	if err != nil {
		h.handleError(fmt.Sprintf("❌ Health check failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		h.handleSuccess(fmt.Sprintf("✅ Health check successful for %s", h.targetEndpoint))
	} else {
		h.handleError(fmt.Sprintf("❌ Health check failed for %s. Status code: %d", h.targetEndpoint, resp.StatusCode))
	}
}

func (h *HealthMonitor) handleSuccess(message string) {
	h.logger.Println(message)
	if err := h.sendSlackMessage(message, false); err != nil {
		h.logger.Printf("Slack notification error: %v", err)
	}
	h.errorCount = 0
	time.Sleep(h.minInterval)
}

func (h *HealthMonitor) handleError(message string) {
	h.logger.Println(message)
	if err := h.sendSlackMessage(message, true); err != nil {
		h.logger.Printf("Slack notification error: %v", err)
	}
	backoffTime := calculateBackoffTime(h.errorCount, h.minInterval, h.maxInterval)
	time.Sleep(backoffTime)
	h.errorCount++
}

func (h *HealthMonitor) StartMonitoring() {
	h.logger.Printf("Monitoring %s", h.targetEndpoint)
	for {
		h.checkHealth()
	}
}

func main() {
	var endpoint, webhookURL string
	var minInterval, maxInterval time.Duration

	rootCmd := &cobra.Command{
		Use:   "health-monitor",
		Short: "Continuous health monitoring CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			if endpoint == "" || webhookURL == "" {
				fmt.Println("Error: endpoint and webhook URL are required")
				os.Exit(1)
			}

			monitor := NewHealthMonitor(endpoint, webhookURL, minInterval, maxInterval)
			monitor.StartMonitoring()
		},
	}

	rootCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "Target endpoint to monitor")
	rootCmd.Flags().StringVarP(&webhookURL, "webhook", "w", "", "Slack webhook URL")
	rootCmd.Flags().DurationVarP(&minInterval, "min-interval", "m", 60*time.Second, "Minimum check interval")
	rootCmd.Flags().DurationVarP(&maxInterval, "max-interval", "M", 3600*time.Second, "Maximum check interval")
	rootCmd.MarkFlagRequired("endpoint")
	rootCmd.MarkFlagRequired("webhook")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

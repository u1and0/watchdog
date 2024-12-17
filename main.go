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
	targetEndpoint   string
	slackWebhookURL  string
	checkInterval    time.Duration
	maxRetryInterval time.Duration
	logger           *log.Logger
}

func NewHealthMonitor(endpoint, webhookURL string, interval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		targetEndpoint:   endpoint,
		slackWebhookURL:  webhookURL,
		checkInterval:    interval,
		maxRetryInterval: 1 * time.Hour,
		logger:           log.New(os.Stdout, "HealthMonitor: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func calculateBackoffTime(errorCount int, baseInterval, maxInterval time.Duration) time.Duration {
	backoffTime := baseInterval * time.Duration(1<<errorCount)
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
	client := &http.Client{Timeout: 10 * time.Second}
	errorCount := 0

	resp, err := client.Get(h.targetEndpoint)
	if err != nil {
		errorMsg := fmt.Sprintf("❌ Health check failed: %v", err)
		h.logger.Println(errorMsg)
		if errSlack := h.sendSlackMessage(errorMsg, true); errSlack != nil {
			h.logger.Printf("Slack notification error: %v", errSlack)
		}

		// バックオフ時間の計算
		backoffTime := calculateBackoffTime(errorCount, h.checkInterval, h.maxRetryInterval)
		time.Sleep(backoffTime)
		errorCount++
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		successMsg := fmt.Sprintf("✅ Health check successful for %s", h.targetEndpoint)
		h.logger.Println(successMsg)
		if err := h.sendSlackMessage(successMsg, false); err != nil {
			h.logger.Printf("Slack notification error: %v", err)
		}
		time.Sleep(h.checkInterval)
	} else {
		errorMsg := fmt.Sprintf("❌ Health check failed for %s. Status code: %d", h.targetEndpoint, resp.StatusCode)
		h.logger.Println(errorMsg)
		if err := h.sendSlackMessage(errorMsg, true); err != nil {
			h.logger.Printf("Slack notification error: %v", err)
		}

		// バックオフ時間の計算
		backoffTime := calculateBackoffTime(errorCount, h.checkInterval, h.maxRetryInterval)
		time.Sleep(backoffTime)
		errorCount++
	}
}

func (h *HealthMonitor) StartMonitoring() {
	h.logger.Printf("Monitoring %s", h.targetEndpoint)
	for {
		h.checkHealth()
	}
}

func main() {
	var endpoint, webhookURL string
	var checkInterval time.Duration

	rootCmd := &cobra.Command{
		Use:   "health-monitor",
		Short: "Continuous health monitoring CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			if endpoint == "" || webhookURL == "" {
				fmt.Println("Error: endpoint and webhook URL are required")
				os.Exit(1)
			}

			monitor := NewHealthMonitor(endpoint, webhookURL, checkInterval)
			monitor.StartMonitoring()
		},
	}

	rootCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "Target endpoint to monitor")
	rootCmd.Flags().StringVarP(&webhookURL, "webhook", "w", "", "Slack webhook URL")
	rootCmd.Flags().DurationVarP(&checkInterval, "interval", "i", 60*time.Second, "Check interval")
	rootCmd.MarkFlagRequired("endpoint")
	rootCmd.MarkFlagRequired("webhook")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

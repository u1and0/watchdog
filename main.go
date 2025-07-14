/*
`watchdog` is a continuous health monitoring CLI tool

usage:

	$ watchdog -e http://localhost:8080 -m 2s -M 16s -w $SLACK_WEBHOOK -u $SLACK_USER
*/
package main

import (
	"fmt"
	"io"
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
	slackUserID     string
	minInterval     time.Duration
	maxInterval     time.Duration
	logger          *log.Logger
	client          *http.Client
	errorCount      int
	lastSuccessTime time.Time
}

func NewHealthMonitor(endpoint, webhookURL, userID string, minInterval, maxInterval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		targetEndpoint:  endpoint,
		slackWebhookURL: webhookURL,
		slackUserID:     userID,
		minInterval:     minInterval,
		maxInterval:     maxInterval,
		logger:          log.New(os.Stdout, "HealthMonitor: ", log.Ldate|log.Ltime|log.Lshortfile),
		client:          &http.Client{Timeout: 10 * time.Second},
		errorCount:      0,
		lastSuccessTime: time.Now().Add(-maxInterval),
	}
}

func (h *HealthMonitor) sendSlackMessage(message string, isError bool) error {
	color := "good"
	if isError {
		color = "danger"
		// Errorの場合、userIDが指定されていればリプライ
		if h.slackUserID != "" {
			message = fmt.Sprintf("<@%s> %s", h.slackUserID, message)
		}
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
		h.handleFailure(fmt.Sprintf("❌ Health check failed: %v", err))
		return
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		h.handleFailure(fmt.Sprintf("❌ Health check failed: %v", err))
		return
	}
	// slackに正常メッセージを送るか、エラーメッセージを送るか。
	// エラーメッセージの場合は、再帰的に
	// 指数バックオフを使って時間差でリトライする処理へ移る。
	if resp.StatusCode != http.StatusOK {
		h.handleFailure(string(b))
		return // minInterval の sleep は handleFailure 内で処理
	}
	h.handleSuccess(string(b))
	time.Sleep(h.minInterval)
}

func (h *HealthMonitor) handleSuccess(body string) {
	// 通知が多くなりすぎるため、maxInterval 未満だと何も通知しない
	if time.Since(h.lastSuccessTime) < h.maxInterval {
		return
	}
	// 成功したらHealthMonitorの状態をリセット
	h.errorCount = 0
	h.lastSuccessTime = time.Now()
	// 成功メッセージをSlackに投げる
	message := fmt.Sprintf("✅ Health check successful for %s: %s", h.targetEndpoint, body)
	h.logger.Println(message)
	if err := h.sendSlackMessage(message, false); err != nil {
		h.logger.Printf("Slack notification error: %v", err)
	}
}

func (h *HealthMonitor) handleFailure(body string) {
	// エラーメッセージの作成とログ出力
	message := fmt.Sprintf("❌ Health check failed for %s. %s", h.targetEndpoint, body)
	h.logger.Println(message)
	// Slack通知
	if err := h.sendSlackMessage(message, true); err != nil {
		h.logger.Printf("Slack notification error: %v", err)
	}
	// 指数バックオフを使って時間差でリトライ
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

func calculateBackoffTime(errorCount int, minInterval, maxInterval time.Duration) time.Duration {
	backoffTime := minInterval * time.Duration(1<<uint(errorCount))
	if backoffTime > maxInterval {
		backoffTime = maxInterval
	}
	return backoffTime
}

func main() {
	var endpoint, webhookURL, userID string
	var minInterval, maxInterval time.Duration
	var ok bool

	rootCmd := &cobra.Command{
		Use:   "health-monitor",
		Short: "Continuous health monitoring CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			if endpoint == "" {
				fmt.Println("Error: endpoint is required")
				os.Exit(1)
			}
			if webhookURL == "" {
				webhookURL, ok = os.LookupEnv("SLACK_WEBHOOK")
				if !ok {
					fmt.Println("Error: webhook URL is required")
				}
				os.Exit(1)
			}
			// User is optional, if exist reply on slack
			if userID == "" {
				userID = os.Getenv("SLACK_USER")
			}

			monitor := NewHealthMonitor(endpoint, webhookURL, userID, minInterval, maxInterval)
			monitor.StartMonitoring()
		},
	}

	rootCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "Target endpoint to monitor")
	rootCmd.Flags().StringVarP(&webhookURL, "webhook", "w", "", "Slack webhook URL")
	rootCmd.Flags().StringVarP(&userID, "user", "u", "", "Slack user ID to mention in error messages")
	rootCmd.Flags().DurationVarP(&minInterval, "min-interval", "m", 60*time.Second, "Minimum check interval")
	rootCmd.Flags().DurationVarP(&maxInterval, "max-interval", "M", 3600*time.Second, "Maximum check interval")
	rootCmd.MarkFlagRequired("endpoint")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

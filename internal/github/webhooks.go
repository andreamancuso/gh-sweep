package github

import "fmt"

// Webhook represents a repository webhook
type Webhook struct {
	ID         int
	Repository string
	URL        string
	Events     []string
	Active     bool
}

type webhookResponse struct {
	ID     int    `json:"id"`
	URL    string `json:"url"`
	Config struct {
		URL string `json:"url"`
	} `json:"config"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

// ListWebhooks lists all webhooks for a repository
func (c *Client) ListWebhooks(owner, repo string) ([]Webhook, error) {
	var response []webhookResponse
	path := apiPath("repos", owner, repo, "hooks")

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	webhooks := make([]Webhook, len(response))
	for i, w := range response {
		webhooks[i] = Webhook{
			ID:         w.ID,
			Repository: repoFullName(owner, repo),
			URL:        w.Config.URL,
			Events:     w.Events,
			Active:     w.Active,
		}
	}

	return webhooks, nil
}

// WebhookDelivery represents a webhook delivery
type WebhookDelivery struct {
	ID        int
	Event     string
	Status    int
	Duration  int // milliseconds
	Timestamp string
}

type deliveryResponse struct {
	ID        int    `json:"id"`
	Event     string `json:"event"`
	Status    int    `json:"status_code"`
	Duration  int    `json:"duration"`
	Delivered string `json:"delivered_at"`
}

// ListWebhookDeliveries lists recent deliveries for a webhook
func (c *Client) ListWebhookDeliveries(owner, repo string, hookID int) ([]WebhookDelivery, error) {
	var response []deliveryResponse
	path := apiPath("repos", owner, repo, "hooks", fmt.Sprintf("%d", hookID), "deliveries")

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list webhook deliveries: %w", err)
	}

	deliveries := make([]WebhookDelivery, len(response))
	for i, d := range response {
		deliveries[i] = WebhookDelivery{
			ID:        d.ID,
			Event:     d.Event,
			Status:    d.Status,
			Duration:  d.Duration,
			Timestamp: d.Delivered,
		}
	}

	return deliveries, nil
}

// WebhookHealth represents webhook health metrics
type WebhookHealth struct {
	WebhookID       int
	SuccessRate     float64
	TotalDeliveries int
	Failures        int
	AvgDuration     int
}

// AnalyzeWebhookHealth analyzes webhook delivery health
func AnalyzeWebhookHealth(deliveries []WebhookDelivery) WebhookHealth {
	health := WebhookHealth{
		TotalDeliveries: len(deliveries),
	}

	if len(deliveries) == 0 {
		return health
	}

	successCount := 0
	totalDuration := 0

	for _, d := range deliveries {
		if d.Status >= 200 && d.Status < 300 {
			successCount++
		} else {
			health.Failures++
		}
		totalDuration += d.Duration
	}

	health.SuccessRate = float64(successCount) / float64(len(deliveries)) * 100
	health.AvgDuration = totalDuration / len(deliveries)

	return health
}

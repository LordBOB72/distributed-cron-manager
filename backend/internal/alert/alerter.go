package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Alerter struct {
	webhookURL string
	client     *http.Client
	log        *zap.Logger
}

func New(webhookURL string, log *zap.Logger) *Alerter {
	return &Alerter{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 8 * time.Second},
		log:        log,
	}
}

func (a *Alerter) Send(ctx context.Context, msg string) {
	if a.webhookURL == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{"text": msg})
	for attempt := 1; attempt <= 3; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.webhookURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := a.client.Do(req)
		if err == nil && resp.StatusCode < 300 {
			resp.Body.Close()
			return
		}
		if err != nil {
			a.log.Warn("alert attempt failed", zap.Int("attempt", attempt), zap.Error(err))
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
}

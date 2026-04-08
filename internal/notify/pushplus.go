package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type PushPlus struct {
	Token  string
	Client *http.Client
}

func (p PushPlus) Name() string {
	return "pushplus"
}

func (p PushPlus) Send(ctx context.Context, msg Message) error {
	token := strings.TrimSpace(p.Token)
	if token == "" {
		return errors.New("empty pushplus token")
	}

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	body := map[string]string{
		"token":    token,
		"title":    msg.Title,
		"content":  msg.Body,
		"template": "txt",
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.pushplus.plus/send", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("pushplus returned non-2xx status")
	}
	return nil
}

package handlers

import (
	"fmt"
	"net/http"

	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/utils"
)

type Webhook struct {
	URL string `json:"url"`
}

func SendWebhooks(channel courier.Channel, r *http.Request, webhook interface{}) error {
	webhookURL, ok := webhook.(map[string]string)
	if !ok {
		return fmt.Errorf("")
	}
	req, err := http.NewRequest(http.MethodPost, webhookURL["url"], r.Body)
	if err != nil {
		return err
	}

	resp, err := utils.MakeHTTPRequest(req)
	if err != nil {
		return err
	}

	if resp.StatusCode/100 != 2 {
		return err
	}

	return nil
}

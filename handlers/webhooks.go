package handlers

import (
	"net/http"

	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/utils"
)

func SendWebhooks(channel courier.Channel, r *http.Request, webhook string) error {
	req, err := http.NewRequest(http.MethodPost, webhook, r.Body)
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

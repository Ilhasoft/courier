package messangi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/utils/clogs"
	"github.com/nyaruka/gocommon/urns"
)

func init() {
	courier.RegisterHandler(newHandler())
}

type handler struct {
	handlers.BaseHandler
}

func newHandler() courier.ChannelHandler {
	return &handler{handlers.NewBaseHandler(courier.ChannelType("MG"), "Messangi")}
}

type moPayload struct {
	Owner      string `json:"owner"`
	Date       string `json:"date"`
	ProcessID  string `json:"processId"`
	Origin     string `json:"origin"`
	ExternalID string `json:"externalId"`
	Callback   string `json:"callback"`
	Connection string `json:"connection"`
	ID         string `json:"id"`
	Text       string `json:"text"`
	User       string `json:"user"`
	ExtraInfo  any    `json:"extraInfo"`
}

// Initialize is called by the engine once everything is loaded
func (h *handler) Initialize(s courier.Server) error {
	h.SetServer(s)
	receiveHandler := handlers.JSONPayload(h, h.receiveMessage)
	s.AddHandlerRoute(h, http.MethodPost, "receive", courier.ChannelLogTypeMsgReceive, receiveHandler)
	return nil
}

func (h *handler) receiveMessage(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request, payload *moPayload, clog *courier.ChannelLog) ([]courier.Event, error) {
	// validate required fields
	if payload.User == "" {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, fmt.Errorf("missing required field 'user'"))
	}
	if payload.Text == "" {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, fmt.Errorf("missing required field 'text'"))
	}

	// parse the date
	date := time.Now()
	if payload.Date != "" {
		if parsedDate, err := time.Parse(time.RFC3339, payload.Date); err == nil {
			date = parsedDate
		}
	}

	// create our URN
	urn, err := urns.ParsePhone(payload.User, channel.Country(), true, false)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// create our message
	msg := h.Backend().NewIncomingMsg(ctx, channel, urn, payload.Text, payload.ID, clog).WithReceivedOn(date)

	// and finally write our message
	return handlers.WriteMsgsAndResponse(ctx, h, []courier.MsgIn{msg}, w, r, clog)
}

func (h *handler) Send(ctx context.Context, msg courier.MsgOut, res *courier.SendResult, clog *courier.ChannelLog) error {
	// get our access token
	accessToken := msg.Channel().StringConfigForKey(courier.ConfigAuthToken, "")
	if accessToken == "" {
		return courier.ErrChannelConfig
	}

	// build our request
	form := map[string]interface{}{
		"from": msg.Channel().Address(),
		"to":   strings.TrimPrefix(msg.URN().Path(), "+"),
		"text": msg.Text(),
		"type": "MT",
	}

	// build our URL and request
	url := "https://elastic.messangi.me/raven/v2/messages"
	jsonBody, err := json.Marshal(form)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, respBody, err := h.RequestHTTP(req, clog)
	if err != nil || resp.StatusCode/100 == 5 {
		return courier.ErrConnectionFailed
	} else if resp.StatusCode/100 != 2 {
		return courier.ErrResponseStatus
	}

	// parse our response
	responseData := &struct {
		Status      string `json:"status"`
		MessageID   string `json:"messageId"`
		Description string `json:"description"`
	}{}

	err = json.Unmarshal(respBody, responseData)
	if err != nil {
		clog.Error(&clogs.Error{Code: "response_unparseable", Message: "Unable to parse response body from Messangi"})
		return courier.ErrResponseUnparseable
	}

	// check if message was accepted and we have a message ID
	if responseData.Status == "ACCEPTED" && responseData.MessageID != "" {
		res.AddExternalID(responseData.MessageID)
		return nil
	}

	// this was a failure, log the description if available
	if responseData.Description != "" {
		clog.Error(&clogs.Error{Code: "messangi_error", Message: fmt.Sprintf("Messangi API error: %s", responseData.Description)})
	} else {
		clog.Error(&clogs.Error{Code: "message_not_accepted", Message: "Message not accepted by Messangi"})
	}
	return courier.ErrResponseStatus
}

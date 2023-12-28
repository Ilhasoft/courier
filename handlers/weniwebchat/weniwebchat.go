package weniwebchat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/gocommon/urns"
	"github.com/sirupsen/logrus"
)

func init() {
	courier.RegisterHandler(newHandler())
}

type handler struct {
	handlers.BaseHandler
}

func newHandler() courier.ChannelHandler {
	return &handler{handlers.NewBaseHandler(courier.ChannelType("WWC"), "Weni Web Chat")}
}

// Initialize is called by the engine once everything is loaded
func (h *handler) Initialize(s courier.Server) error {
	h.SetServer(s)
	s.AddHandlerRoute(h, http.MethodPost, "receive", h.receiveEvent)
	return nil
}

type miPayload struct {
	Type    string    `json:"type"           validate:"required"`
	From    string    `json:"from,omitempty" validate:"required"`
	Message miMessage `json:"message"`
}

type miMessage struct {
	Type      string `json:"type"          validate:"required"`
	TimeStamp string `json:"timestamp"     validate:"required"`
	Text      string `json:"text,omitempty"`
	MediaURL  string `json:"media_url,omitempty"`
	Caption   string `json:"caption,omitempty"`
	Latitude  string `json:"latitude,omitempty"`
	Longitude string `json:"longitude,omitempty"`
}

func (h *handler) receiveEvent(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request, clog *courier.ChannelLog) ([]courier.Event, error) {
	payload := &miPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// check message type
	if payload.Type != "message" || (payload.Message.Type != "text" && payload.Message.Type != "image" && payload.Message.Type != "video" && payload.Message.Type != "audio" && payload.Message.Type != "file" && payload.Message.Type != "location") {
		return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "ignoring request, unknown message type")
	}

	// check empty content
	if payload.Message.Text == "" && payload.Message.MediaURL == "" && (payload.Message.Latitude == "" || payload.Message.Longitude == "") {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errors.New("blank message, media or location"))
	}

	// build urn
	urn, err := urns.NewURNFromParts(urns.ExternalScheme, payload.From, "", "")
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// parse timestamp
	ts, err := strconv.ParseInt(payload.Message.TimeStamp, 10, 64)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, fmt.Errorf("invalid timestamp: %s", payload.Message.TimeStamp))
	}

	// parse medias
	var mediaURL string
	if payload.Message.Type == "location" {
		mediaURL = fmt.Sprintf("geo:%s,%s", payload.Message.Latitude, payload.Message.Longitude)
	} else if payload.Message.MediaURL != "" {
		mediaURL = payload.Message.MediaURL
		payload.Message.Text = payload.Message.Caption
	}

	// build message
	date := time.Unix(ts, 0).UTC()
	msg := h.Backend().NewIncomingMsg(channel, urn, payload.Message.Text, clog).WithReceivedOn(date).WithContactName(payload.From)

	if mediaURL != "" {
		msg.WithAttachment(mediaURL)
	}

	return handlers.WriteMsgsAndResponse(ctx, h, []courier.Msg{msg}, w, r, clog)
}

var timestamp = ""

type moPayload struct {
	Type    string    `json:"type" validate:"required"`
	To      string    `json:"to"   validate:"required"`
	From    string    `json:"from" validate:"required"`
	Message moMessage `json:"message"`
}

type moMessage struct {
	Type         string   `json:"type"      validate:"required"`
	TimeStamp    string   `json:"timestamp" validate:"required"`
	Text         string   `json:"text,omitempty"`
	MediaURL     string   `json:"media_url,omitempty"`
	Caption      string   `json:"caption,omitempty"`
	Latitude     string   `json:"latitude,omitempty"`
	Longitude    string   `json:"longitude,omitempty"`
	QuickReplies []string `json:"quick_replies,omitempty"`
}

func (h *handler) Send(ctx context.Context, msg courier.Msg, clog *courier.ChannelLog) (courier.MsgStatus, error) {
	status := h.Backend().NewMsgStatusForID(msg.Channel(), msg.ID(), courier.MsgSent, clog)

	baseURL := msg.Channel().StringConfigForKey(courier.ConfigBaseURL, "")
	if baseURL == "" {
		return nil, errors.New("blank base_url")
	}

	sendURL := fmt.Sprintf("%s/send", baseURL)

	payload := newOutgoingMessage("message", msg.URN().Path(), msg.Channel().Address(), msg.QuickReplies())
	lenAttachments := len(msg.Attachments())
	if lenAttachments > 0 {

	attachmentsLoop:
		for i, attachment := range msg.Attachments() {
			mimeType, attachmentURL := handlers.SplitAttachment(attachment)
			payload.Message.TimeStamp = getTimestamp()
			// parse attachment type
			if strings.HasPrefix(mimeType, "audio") {
				payload.Message = moMessage{
					Type:     "audio",
					MediaURL: attachmentURL,
				}
			} else if strings.HasPrefix(mimeType, "application") {
				payload.Message = moMessage{
					Type:     "file",
					MediaURL: attachmentURL,
				}
			} else if strings.HasPrefix(mimeType, "image") {
				payload.Message = moMessage{
					Type:     "image",
					MediaURL: attachmentURL,
				}
			} else if strings.HasPrefix(mimeType, "video") {
				payload.Message = moMessage{
					Type:     "video",
					MediaURL: attachmentURL,
				}
			} else {
				logrus.WithField("channel_uuid", msg.Channel().UUID().String()).Error("unknown attachment mime type: ", mimeType)
				status.SetStatus(courier.MsgFailed)
				break attachmentsLoop
			}

			// add a caption to the first attachment
			if i == 0 {
				payload.Message.Caption = msg.Text()
			}

			// add quickreplies on last message
			if i == lenAttachments-1 {
				payload.Message.QuickReplies = msg.QuickReplies()
			}

			// build request
			var body []byte
			body, err := json.Marshal(&payload)
			if err != nil {
				logrus.WithField("channel_uuid", msg.Channel().UUID().String()).WithError(err).Error("Error sending message")
				status.SetStatus(courier.MsgFailed)
				break attachmentsLoop
			}
			req, _ := http.NewRequest(http.MethodPost, sendURL, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			_, _, err = handlers.RequestHTTP(req, clog)
			if err != nil {
				logrus.WithField("channel_uuid", msg.Channel().UUID().String()).WithError(err).Error("Message Send Error")
				status.SetStatus(courier.MsgFailed)
			}
			if err != nil {
				status.SetStatus(courier.MsgFailed)
				break attachmentsLoop
			}
		}
	} else {
		payload.Message = moMessage{
			Type:         "text",
			TimeStamp:    getTimestamp(),
			Text:         msg.Text(),
			QuickReplies: msg.QuickReplies(),
		}
		// build request
		body, err := json.Marshal(&payload)
		if err != nil {
			logrus.WithField("channel_uuid", msg.Channel().UUID().String()).WithError(err).Error("Message Send Error")
			status.SetStatus(courier.MsgFailed)
			status.SetStatus(courier.MsgFailed)
		} else {
			req, _ := http.NewRequest(http.MethodPost, sendURL, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			_, _, err := handlers.RequestHTTP(req, clog)
			if err != nil {
				status.SetStatus(courier.MsgFailed)
			}
		}

	}

	return status, nil
}

func newOutgoingMessage(payType, to, from string, quickReplies []string) *moPayload {
	return &moPayload{
		Type: payType,
		To:   to,
		From: from,
		Message: moMessage{
			QuickReplies: quickReplies,
		},
	}
}

func getTimestamp() string {
	if timestamp != "" {
		return timestamp
	}

	return fmt.Sprint(time.Now().Unix())
}

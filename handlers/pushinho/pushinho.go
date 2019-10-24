package pushinho

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/utils"
	"github.com/nyaruka/gocommon/urns"
	"github.com/pkg/errors"
)

var (
	sendURL    = "https://socket.push.al"
	maxMsgSize = 1024
)

const pushinhoScheme = "ps"

type handler struct {
	handlers.BaseHandler
}

type quickReply struct {
	Title string `json:"title"`
}

type sendForm struct {
	To       string `json:"to" validate:"required"`
	Text     string `json:"text"`
	Metadata struct {
		QuickReplies []quickReply `json:"quick_replies,omitempty"`
	} `json:"metadata"`
}

type rawMessage struct {
	To   string `json:"to" validate:"required"`
	Text string `json:"text"`
}

type receiveForm struct {
	From string `json:"from"       validate:"required"`
	Text string `json:"text"`
}

// NewPushinhoURN returns a URN for the passed in pushinho identifier
func NewPushinhoURN(identifier string) (urns.URN, error) {
	urns.ValidSchemes[pushinhoScheme] = true
	return urns.NewURNFromParts(pushinhoScheme, identifier, "", "")
}

func (h *handler) receiveMessage(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	form := &receiveForm{}

	// decode the post from the user
	err := handlers.DecodeAndValidateForm(form, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// URN now created
	urn, err := NewPushinhoURN(form.From)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// creating message
	dbMsg := h.Backend().NewIncomingMsg(channel, urn, form.Text)
	return handlers.WriteMsgsAndResponse(ctx, h, []courier.Msg{dbMsg}, w, r)
}

func init() {
	courier.RegisterHandler(newHandler())
}

func newHandler() courier.ChannelHandler {
	return &handler{handlers.NewBaseHandler(courier.ChannelType("PS"), "Pushinho")}
}

func (h *handler) Initialize(s courier.Server) error {
	h.SetServer(s)
	s.AddHandlerRoute(h, http.MethodPost, "receive", h.receiveMessage)
	return nil
}

func (h *handler) SendMsg(ctx context.Context, msg courier.Msg) (courier.MsgStatus, error) {
	form := &sendForm{}
	form.To = urns.URN(msg.URN()).String()
	str := msg.Text()
	form.Text = str

	// check if QuickReplies is empty
	if len(msg.Metadata()) > 0 {
		var metadataJson map[string][]quickReply
		err := json.Unmarshal(msg.Metadata(), &metadataJson)
		if err != nil {
			fmt.Println(err)
			log.Fatalln(err)
		}
		for _, reply := range metadataJson["quick_replies"] {
			form.Metadata.QuickReplies = append(form.Metadata.QuickReplies, reply)
		}
	}

	// check if there were any errors while Marshalling json
	body, err := json.Marshal(form)
	if err != nil {
		log.Fatalln(err)
	}

	// creating the request.
	req, _ := http.NewRequest("POST", sendURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// requesting
	response, err := utils.MakeHTTPRequest(req)
	status := h.Backend().NewMsgStatusForID(msg.Channel(), msg.ID(), courier.MsgErrored)

	// logging (expects error, as default)
	log := courier.NewChannelLogFromRR("Message Sent", msg.Channel(), msg.ID(), response).WithError("Message Send Error", err)
	status.AddLog(log)
	if err != nil {
		return status, err
	}

	// checking for success
	if string(response.Body) != "success" {
		err = errors.Errorf("Failed request!")
		log.WithError("Message Send Error", err)
		return status, err
	}
	status.SetStatus(courier.MsgWired)
	return status, nil
}

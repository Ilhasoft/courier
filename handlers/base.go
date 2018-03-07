package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/schema"
	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/utils"
	"github.com/nyaruka/gocommon/urns"
	validator "gopkg.in/go-playground/validator.v9"
)

var base64Regex, _ = regexp.Compile("^([a-zA-Z0-9+/=]{4})+$")
var base64Encoding = base64.StdEncoding.Strict()

// BaseHandler is the base class for most handlers, it just stored the server, name and channel type for the handler
type BaseHandler struct {
	channelType courier.ChannelType
	name        string
	server      courier.Server
	backend     courier.Backend
}

// NewBaseHandler returns a newly constructed BaseHandler with the passed in parameters
func NewBaseHandler(channelType courier.ChannelType, name string) BaseHandler {
	return BaseHandler{channelType: channelType, name: name}
}

// SetServer can be used to change the server on a BaseHandler
func (h *BaseHandler) SetServer(server courier.Server) {
	h.server = server
	h.backend = server.Backend()
}

// Server returns the server instance on the BaseHandler
func (h *BaseHandler) Server() courier.Server {
	return h.server
}

// Backend returns the backend instance on the BaseHandler
func (h *BaseHandler) Backend() courier.Backend {
	return h.backend
}

// ChannelType returns the channel type that this handler deals with
func (h *BaseHandler) ChannelType() courier.ChannelType {
	return h.channelType
}

// ChannelName returns the name of the channel this handler deals with
func (h *BaseHandler) ChannelName() string {
	return h.name
}

// ResponseSuccess interace with response methods for success responses
type ResponseSuccess interface {
	Backend() courier.Backend
	WriteStatusSuccessResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, statuses []courier.MsgStatus) error
	WriteMsgSuccessResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, msgs []courier.Msg) error
}

// WriteStatusSuccessResponse writes a success response for the statuses
func (h *BaseHandler) WriteStatusSuccessResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, statuses []courier.MsgStatus) error {
	return courier.WriteStatusSuccess(ctx, w, r, statuses)
}

// WriteMsgSuccessResponse writes a success response for the messages
func (h *BaseHandler) WriteMsgSuccessResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, msgs []courier.Msg) error {
	return courier.WriteMsgSuccess(ctx, w, r, msgs)
}

var (
	decoder  = schema.NewDecoder()
	validate = validator.New()
)

func init() {
	decoder.IgnoreUnknownKeys(true)
	decoder.SetAliasTag("name")
}

// NameFromFirstLastUsername is a utility function to build a contact's name from the passed
// in values, all of which can be empty
func NameFromFirstLastUsername(first string, last string, username string) string {
	if first != "" && last != "" {
		return fmt.Sprintf("%s %s", first, last)
	} else if first != "" {
		return first
	} else if last != "" {
		return last
	} else if username != "" {
		return username
	}
	return ""
}

// Validate validates the passe din struct using our shared validator instance
func Validate(form interface{}) error {
	return validate.Struct(form)
}

// DecodeAndValidateForm takes the passed in form and attempts to parse and validate it from the
// URL query parameters as well as any POST parameters of the passed in request
func DecodeAndValidateForm(form interface{}, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = decoder.Decode(form, r.Form)
	if err != nil {
		return err
	}

	// check our input is valid
	err = validate.Struct(form)
	if err != nil {
		return err
	}

	return nil
}

// DecodeAndValidateJSON takes the passed in envelope and tries to unmarshal it from the body
// of the passed in request, then validating it
func DecodeAndValidateJSON(envelope interface{}, r *http.Request) error {
	// read our body
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 100000))
	defer r.Body.Close()
	if err != nil {
		return fmt.Errorf("unable to read request body: %s", err)
	}

	// try to decode our envelope
	if err = json.Unmarshal(body, envelope); err != nil {
		return fmt.Errorf("unable to parse request JSON: %s", err)
	}

	// check our input is valid
	err = validate.Struct(envelope)
	if err != nil {
		return fmt.Errorf("request JSON doesn't match required schema: %s", err)
	}

	return nil
}

// DecodeAndValidateXML takes the passed in envelope and tries to unmarshal it from the body
// of the passed in request, then validating it
func DecodeAndValidateXML(envelope interface{}, r *http.Request) error {
	// read our body
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 100000))
	defer r.Body.Close()
	if err != nil {
		return fmt.Errorf("unable to read request body: %s", err)
	}

	// try to decode our envelope
	if err = xml.Unmarshal(body, envelope); err != nil {
		return fmt.Errorf("unable to parse request XML: %s", err)
	}

	// check our input is valid
	err = validate.Struct(envelope)
	if err != nil {
		return fmt.Errorf("request XML doesn't match required schema: %s", err)
	}

	return nil
}

/*
DecodePossibleBase64 detects and decodes a possibly base64 encoded messages by doing:
 * check it's at least 60 characters
 * check its length is divisible by 4
 * check that there's no whitespace
 * check the decoded string contains at least 50% ascii
*/
func DecodePossibleBase64(original string) string {
	stripped := strings.TrimSpace(strings.Replace(strings.Replace(original, "\r", "", -1), "\n", "", -1))
	length := len([]rune(stripped))

	if length < 60 || length%4 != 0 {
		return original
	}

	if !base64Regex.MatchString(stripped[:length-4]) {
		return original
	}

	decodedBytes, err := base64Encoding.DecodeString(stripped)
	if err != nil {
		return original
	}

	decoded := utils.DecodeUTF8(decodedBytes)
	numASCIIChars := 0
	for _, c := range decoded {
		if c <= 127 {
			numASCIIChars++
		}
	}

	if float32(numASCIIChars)/float32(len([]rune(decoded))) < 0.5 {
		return original
	}

	return decoded
}

// SplitMsg splits the passed in string into segments that are at most max length
func SplitMsg(text string, max int) []string {
	// smaller than our max, just return it
	if len(text) <= max {
		return []string{text}
	}

	parts := make([]string, 0, 2)
	part := bytes.Buffer{}
	for _, r := range text {
		part.WriteRune(r)
		if part.Len() == max || (part.Len() > max-6 && r == ' ') {
			parts = append(parts, strings.TrimSpace(part.String()))
			part.Reset()
		}
	}
	if part.Len() > 0 {
		parts = append(parts, strings.TrimSpace(part.String()))
	}

	return parts
}

// NewTelReceiveHandler creates a new receive handler given the passed in text and from fields
func NewTelReceiveHandler(h BaseHandler, fromField string, bodyField string) courier.ChannelHandleFunc {
	return func(ctx context.Context, c courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
		err := r.ParseForm()
		if err != nil {
			return nil, courier.WriteAndLogRequestError(ctx, w, r, c, err)
		}

		body := r.Form.Get(bodyField)
		from := r.Form.Get(fromField)
		if from == "" {
			return nil, courier.WriteAndLogRequestError(ctx, w, r, c, fmt.Errorf("missing required field '%s'", fromField))
		}
		// create our URN
		urn, err := urns.NewTelURNForCountry(from, c.Country())
		if err != nil {
			return nil, courier.WriteAndLogRequestError(ctx, w, r, c, err)
		}
		// build our msg
		msg := h.Backend().NewIncomingMsg(c, urn, body).WithReceivedOn(time.Now().UTC())

		// and finally queue our message
		err = h.Backend().WriteMsg(ctx, msg)
		if err != nil {
			return nil, err
		}

		return []courier.Event{msg}, courier.WriteMsgSuccess(ctx, w, r, []courier.Msg{msg})
	}
}

// NewExternalIDStatusHandler creates a new status handler given the passed in status map and fields
func NewExternalIDStatusHandler(h BaseHandler, statuses map[string]courier.MsgStatusValue, externalIDField string, statusField string) courier.ChannelHandleFunc {
	return func(ctx context.Context, c courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
		err := r.ParseForm()
		if err != nil {
			return nil, courier.WriteAndLogRequestError(ctx, w, r, c, err)
		}

		externalID := r.Form.Get(externalIDField)
		if externalID == "" {
			return nil, courier.WriteAndLogRequestError(ctx, w, r, c, fmt.Errorf("missing required field '%s'", externalIDField))
		}

		s := r.Form.Get(statusField)
		sValue, found := statuses[s]
		if !found {
			return nil, courier.WriteAndLogRequestError(ctx, w, r, c, fmt.Errorf("unknown status value '%s'", s))
		}

		// create our status
		status := h.Backend().NewMsgStatusForExternalID(c, externalID, sValue)
		err = h.Backend().WriteMsgStatus(ctx, status)
		if err == courier.ErrMsgNotFound {
			return nil, courier.WriteAndLogStatusMsgNotFound(ctx, w, r, c)
		}
		if err != nil {
			return nil, err
		}

		return []courier.Event{status}, courier.WriteStatusSuccess(ctx, w, r, []courier.MsgStatus{status})
	}
}

// GetTextAndAttachments returns both the text of our message as well as any attachments, newline delimited
func GetTextAndAttachments(m courier.Msg) string {
	buf := bytes.NewBuffer([]byte(m.Text()))
	for _, a := range m.Attachments() {
		_, url := SplitAttachment(a)
		buf.WriteString("\n")
		buf.WriteString(url)
	}
	return buf.String()
}

// SplitAttachment takes an attachment string and returns the media type and URL for the attachment
func SplitAttachment(attachment string) (string, string) {
	parts := strings.SplitN(attachment, ":", 2)
	if len(parts) < 2 {
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// WriteMsgAndResponse writes the passed in message to our backend
func WriteMsgAndResponse(ctx context.Context, h ResponseSuccess, msg courier.Msg, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	err := h.Backend().WriteMsg(ctx, msg)
	if err != nil {
		return nil, err
	}

	return []courier.Event{msg}, h.WriteMsgSuccessResponse(ctx, w, r, []courier.Msg{msg})
}

// WriteMsgStatusAndResponse write the passed in status to our backend
func WriteMsgStatusAndResponse(ctx context.Context, h ResponseSuccess, channel courier.Channel, status courier.MsgStatus, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	err := h.Backend().WriteMsgStatus(ctx, status)
	if err == courier.ErrMsgNotFound {
		return nil, courier.WriteAndLogStatusMsgNotFound(ctx, w, r, channel)
	}

	if err != nil {
		return nil, err
	}

	return []courier.Event{status}, h.WriteStatusSuccessResponse(ctx, w, r, []courier.MsgStatus{status})
}

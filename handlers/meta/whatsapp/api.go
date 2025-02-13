package whatsapp

import "github.com/nyaruka/courier"

// see https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/payload-examples#message-status-updates
var StatusMapping = map[string]courier.MsgStatus{
	"sent":      courier.MsgStatusSent,
	"delivered": courier.MsgStatusDelivered,
	"read":      courier.MsgStatusRead,
	"failed":    courier.MsgStatusFailed,
}

var IgnoreStatuses = map[string]bool{
	"deleted": true,
}

// see https://developers.facebook.com/docs/whatsapp/cloud-api/reference/media#example-2
type MOMedia struct {
	Caption  string `json:"caption"`
	Filename string `json:"filename"`
	ID       string `json:"id"`
	Mimetype string `json:"mime_type"`
	SHA256   string `json:"sha256"`
}

type Change struct {
	Field string `json:"field"`
	Value struct {
		MessagingProduct string `json:"messaging_product"`
		Metadata         *struct {
			DisplayPhoneNumber string `json:"display_phone_number"`
			PhoneNumberID      string `json:"phone_number_id"`
		} `json:"metadata"`
		Contacts []struct {
			Profile struct {
				Name string `json:"name"`
			} `json:"profile"`
			WaID string `json:"wa_id"`
		} `json:"contacts"`
		Messages []struct {
			ID        string `json:"id"`
			From      string `json:"from"`
			Timestamp string `json:"timestamp"`
			Type      string `json:"type"`
			Context   *struct {
				Forwarded           bool   `json:"forwarded"`
				FrequentlyForwarded bool   `json:"frequently_forwarded"`
				From                string `json:"from"`
				ID                  string `json:"id"`
			} `json:"context"`
			Text struct {
				Body string `json:"body"`
			} `json:"text"`
			Image    *MOMedia `json:"image"`
			Audio    *MOMedia `json:"audio"`
			Video    *MOMedia `json:"video"`
			Document *MOMedia `json:"document"`
			Voice    *MOMedia `json:"voice"`
			Location *struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
				Name      string  `json:"name"`
				Address   string  `json:"address"`
			} `json:"location"`
			Button *struct {
				Text    string `json:"text"`
				Payload string `json:"payload"`
			} `json:"button"`
			Interactive struct {
				Type        string `json:"type"`
				ButtonReply struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"button_reply,omitempty"`
				ListReply struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"list_reply,omitempty"`
			} `json:"interactive,omitempty"`
			Contacts []struct {
				Name struct {
					FirstName     string `json:"first_name"`
					LastName      string `json:"last_name"`
					FormattedName string `json:"formatted_name"`
				} `json:"name"`
				Phones []struct {
					Phone string `json:"phone"`
					WaID  string `json:"wa_id"`
					Type  string `json:"type"`
				} `json:"phones"`
			} `json:"contacts"`
			Errors []struct {
				Code  int    `json:"code"`
				Title string `json:"title"`
			} `json:"errors"`
		} `json:"messages"`
		Statuses []struct {
			ID           string `json:"id"`
			RecipientID  string `json:"recipient_id"`
			Status       string `json:"status"`
			Timestamp    string `json:"timestamp"`
			Type         string `json:"type"`
			Conversation *struct {
				ID     string `json:"id"`
				Origin *struct {
					Type string `json:"type"`
				} `json:"origin"`
			} `json:"conversation"`
			Pricing *struct {
				PricingModel string `json:"pricing_model"`
				Billable     bool   `json:"billable"`
				Category     string `json:"category"`
			} `json:"pricing"`
			Errors []struct {
				Code  int    `json:"code"`
				Title string `json:"title"`
			} `json:"errors"`
		} `json:"statuses"`
		Errors []struct {
			Code  int    `json:"code"`
			Title string `json:"title"`
		} `json:"errors"`
	} `json:"value"`
}

// see https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#media-messages
type Media struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type Section struct {
	Title string       `json:"title,omitempty"`
	Rows  []SectionRow `json:"rows" validate:"required"`
}

type SectionRow struct {
	ID          string `json:"id" validate:"required"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type Button struct {
	Type  string `json:"type" validate:"required"`
	Reply struct {
		ID    string `json:"id" validate:"required"`
		Title string `json:"title" validate:"required"`
	} `json:"reply" validate:"required"`
}

type Param struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Payload  string `json:"payload,omitempty"`
	Image    *Media `json:"image,omitempty"`
	Document *Media `json:"document,omitempty"`
	Video    *Media `json:"video,omitempty"`
}

type Component struct {
	Type    string   `json:"type"`
	SubType string   `json:"sub_type,omitempty"`
	Index   string   `json:"index,omitempty"`
	Params  []*Param `json:"parameters"`
}

type Text struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url"`
}

type Language struct {
	Policy string `json:"policy"`
	Code   string `json:"code"`
}

// see https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages#template-object
// e.g. https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages#template-messages
type Template struct {
	Name       string       `json:"name"`
	Language   *Language    `json:"language"`
	Components []*Component `json:"components,omitempty"`
}

// see https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages#interactive-object
// e.g. https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages#interactive-messages
type Interactive struct {
	Type   string `json:"type"`
	Header *struct {
		Type     string `json:"type"`
		Text     string `json:"text,omitempty"`
		Video    *Media `json:"video,omitempty"`
		Image    *Media `json:"image,omitempty"`
		Document *Media `json:"document,omitempty"`
	} `json:"header,omitempty"`
	Body struct {
		Text string `json:"text"`
	} `json:"body" validate:"required"`
	Footer *struct {
		Text string `json:"text"`
	} `json:"footer,omitempty"`
	Action *struct {
		Button   string    `json:"button,omitempty"`
		Sections []Section `json:"sections,omitempty"`
		Buttons  []Button  `json:"buttons,omitempty"`
	} `json:"action,omitempty"`
}

// see https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#request-syntax
// e.g. https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages#message-object
type SendRequest struct {
	MessagingProduct string `json:"messaging_product"`
	RecipientType    string `json:"recipient_type"`
	To               string `json:"to"`
	Type             string `json:"type"`

	Text *Text `json:"text,omitempty"`

	Document *Media `json:"document,omitempty"`
	Image    *Media `json:"image,omitempty"`
	Audio    *Media `json:"audio,omitempty"`
	Video    *Media `json:"video,omitempty"`

	Interactive *Interactive `json:"interactive,omitempty"`

	Template *Template `json:"template,omitempty"`
}

// see https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#response-syntax
// e.g. https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages#successful-response
type SendResponse struct {
	Messages []*struct {
		ID string `json:"id"`
	} `json:"messages"`
	Error struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
	Contacts []*struct {
		Input string `json:"input,omitempty"`
		WaID  string `json:"wa_id,omitempty"`
	} `json:"contacts,omitempty"`
}

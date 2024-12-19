package weniwebchat

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/test"
	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/urns"
)

const channelUUID = "8eb23e93-5ecb-45ba-b726-3b064e0c568c"

var testChannels = []courier.Channel{
	test.NewMockChannel(channelUUID, "WWC", "250788383383", "", []string{}, map[string]any{}),
}

// ReceiveMsg test

var receiveURL = fmt.Sprintf("/c/wwc/%s/receive", channelUUID)

const (
	textMsgTemplate = `
	{
		"type":"message",
		"from":%q,
		"message":{
			"type":"text",
			"timestamp":%q,
			"text":%q
		}
	}
	`

	imgMsgTemplate = `
	{
		"type":"message",
		"from":%q,
		"message":{
			"type":"image",
			"timestamp":%q,
			"media_url":%q,
			"caption":%q
		}
	}
	`

	locationMsgTemplate = `
	{
		"type":"message",
		"from":%q,
		"message":{
			"type":"location",
			"timestamp":%q,
			"latitude":%q,
			"longitude":%q
		}
	}
	`

	invalidMsgTemplate = `
	{
		"type":"foo",
		"from":"bar",
		"message": {
			"id":"000001",
			"type":"text",
			"timestamp":"1616586927"
		}
	}
	`
)

var testCases = []IncomingTestCase{
	{
		Label:                "Receive Valid Text Msg",
		URL:                  receiveURL,
		Data:                 fmt.Sprintf(textMsgTemplate, "2345678", "1616586927", "Hello Test!"),
		ExpectedContactName:  Sp("2345678"),
		ExpectedURN:          "ext:2345678",
		ExpectedMsgText:      Sp("Hello Test!"),
		ExpectedRespStatus:   200,
		ExpectedBodyContains: "Accepted",
	},
	{
		Label:                "Receive Valid Media",
		URL:                  receiveURL,
		Data:                 fmt.Sprintf(imgMsgTemplate, "2345678", "1616586927", "https://link.to/image.png", "My Caption"),
		ExpectedContactName:  Sp("2345678"),
		ExpectedURN:          "ext:2345678",
		ExpectedMsgText:      Sp("My Caption"),
		ExpectedAttachments:  []string{"https://link.to/image.png"},
		ExpectedRespStatus:   200,
		ExpectedBodyContains: "Accepted",
	},
	{
		Label:                "Receive Valid Location",
		URL:                  receiveURL,
		Data:                 fmt.Sprintf(locationMsgTemplate, "2345678", "1616586927", "-9.6996104", "-35.7794614"),
		ExpectedContactName:  Sp("2345678"),
		ExpectedURN:          "ext:2345678",
		ExpectedAttachments:  []string{"geo:-9.6996104,-35.7794614"},
		ExpectedRespStatus:   200,
		ExpectedBodyContains: "Accepted",
	},
	{
		Label:              "Receive Invalid JSON",
		URL:                receiveURL,
		Data:               "{}",
		ExpectedRespStatus: 400,
	},
	{
		Label:                "Receive Text Msg With Blank Message Text",
		URL:                  receiveURL,
		Data:                 fmt.Sprintf(textMsgTemplate, "2345678", "1616586927", ""),
		ExpectedRespStatus:   400,
		ExpectedBodyContains: "blank message, media or location",
	},
	{
		Label:                "Receive Invalid Timestamp",
		URL:                  receiveURL,
		Data:                 fmt.Sprintf(textMsgTemplate, "2345678", "foo", "Hello Test!"),
		ExpectedRespStatus:   400,
		ExpectedBodyContains: "invalid timestamp: foo",
	},
	{
		Label:                "Receive Invalid Message",
		URL:                  receiveURL,
		Data:                 invalidMsgTemplate,
		ExpectedRespStatus:   200,
		ExpectedBodyContains: "ignoring request, unknown message type",
	},
}

func TestIncoming(t *testing.T) {
	RunIncomingTestCases(t, testChannels, newHandler(), testCases)
}

func BenchmarkHandler(b *testing.B) {
	RunChannelBenchmarks(b, testChannels, newHandler(), testCases)
}

// SendMsg test

func setSendURL(s *httptest.Server, h courier.ChannelHandler, c courier.Channel, m courier.MsgOut) {
	c.(*test.MockChannel).SetConfig(courier.ConfigBaseURL, s.URL)
	timestamp = "1616700878"
}

func mockAttachmentURLs(mediaServer *httptest.Server, testCases []OutgoingTestCase) []OutgoingTestCase {
	casesWithMockedUrls := make([]OutgoingTestCase, len(testCases))

	for i, testCase := range testCases {
		mockedCase := testCase

		for j, attachment := range testCase.MsgAttachments {
			mockedCase.MsgAttachments[j] = strings.Replace(attachment, "https://foo.bar", mediaServer.URL, 1)
		}
		casesWithMockedUrls[i] = mockedCase
	}
	return casesWithMockedUrls
}

var sendTestCases = []OutgoingTestCase{
	{
		Label:   "Plain Send",
		MsgText: "Simple Message",
		MsgURN:  "ext:371298371241",
		MockResponses: map[string][]*httpx.MockResponse{
			"*/send": {
				httpx.NewMockResponse(200, nil, []byte(``)),
			},
		},
		ExpectedRequests: []ExpectedRequest{
			{
				Body:    `{"type":"message","to":"371298371241","from":"250788383383","message":{"type":"text","timestamp":"1616700878","text":"Simple Message"}}`,
				Headers: map[string]string{"Content-type": "application/json"},
			},
		},
	},
	{
		Label:   "Unicode Send",
		MsgText: "☺",
		MsgURN:  "ext:371298371241",
		MockResponses: map[string][]*httpx.MockResponse{
			"*/send": {
				httpx.NewMockResponse(200, nil, []byte(``)),
			},
		},
		ExpectedRequests: []ExpectedRequest{
			{
				Body:    `{"type":"message","to":"371298371241","from":"250788383383","message":{"type":"text","timestamp":"1616700878","text":"☺"}}`,
				Headers: map[string]string{"Content-type": "application/json"},
			},
		},
	},
	{
		Label:   "invalid Text Send",
		MsgText: "Error",
		MsgURN:  "ext:371298371241",
		MockResponses: map[string][]*httpx.MockResponse{
			"*/send": {
				httpx.NewMockResponse(400, nil, []byte(``)),
			},
		},
		ExpectedRequests: []ExpectedRequest{
			{
				Body:    `{"type":"message","to":"371298371241","from":"250788383383","message":{"type":"text","timestamp":"1616700878","text":"Error"}}`,
				Headers: map[string]string{"Content-type": "application/json"},
			},
		},
	},
	{
		Label:   "No Timestamp Prepare",
		MsgText: "No prepare",
		MsgURN:  "ext:371298371241",
		MockResponses: map[string][]*httpx.MockResponse{
			"*/send": {
				httpx.NewMockResponse(200, nil, []byte(``)),
			},
		},
		ExpectedRequests: []ExpectedRequest{
			{
				Body: ``,
			},
		},
	},
}

func TestOutgoing(t *testing.T) {
	var defaultChannel = []courier.Channel{
		test.NewMockChannel(channelUUID, "WWC", "250788383383", "", []string{urns.External.Prefix}, map[string]any{courier.ConfigBaseURL: "https://foo.bar"}),
	}

	timestamp = "1616700878"

	RunOutgoingTestCases(t, defaultChannel[0], newHandler(), sendTestCases, nil, nil)
}

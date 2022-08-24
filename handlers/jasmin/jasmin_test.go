package jasmin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/test"
)

var (
	receiveURL          = "/c/js/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive/"
	receiveValidMessage = "content=%05v%05nement&coding=0&From=2349067554729&To=2349067554711&id=1001"
	receiveMissingTo    = "content=%05v%05nement&coding=0&From=2349067554729&id=1001"
	invalidURN          = "content=%05v%05nement&coding=0&From=MTN&To=2349067554711&id=1001"

	statusURL       = "/c/js/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/status/"
	statusDelivered = "id=external1&dlvrd=1"
	statusFailed    = "id=external1&err=1"
	statusUnknown   = "id=external1&err=0&dlvrd=0"
)

var testChannels = []courier.Channel{
	test.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "JS", "2020", "US", nil),
}

var handleTestCases = []ChannelHandleTestCase{
	{Label: "Receive Valid Message", URL: receiveURL, Data: receiveValidMessage, ExpectedStatus: 200, ExpectedResponse: "ACK/Jasmin",
		ExpectedMsgText: Sp("événement"), ExpectedURN: Sp("tel:+2349067554729"), ExpectedExternalID: Sp("1001")},
	{Label: "Receive Missing To", URL: receiveURL, Data: receiveMissingTo, ExpectedStatus: 400,
		ExpectedResponse: "field 'to' required"},
	{Label: "Invalid URN", URL: receiveURL, Data: invalidURN, ExpectedStatus: 400,
		ExpectedResponse: "phone number supplied is not a number"},
	{Label: "Status Delivered", URL: statusURL, Data: statusDelivered, ExpectedStatus: 200, ExpectedResponse: "ACK/Jasmin",
		ExpectedMsgStatus: Sp("D"), ExpectedExternalID: Sp("external1")},
	{Label: "Status Failed", URL: statusURL, Data: statusFailed, ExpectedStatus: 200, ExpectedResponse: "ACK/Jasmin",
		ExpectedMsgStatus: Sp("F"), ExpectedExternalID: Sp("external1")},
	{Label: "Status Missing", URL: statusURL, ExpectedStatus: 400, Data: "nothing",
		ExpectedResponse: "field 'id' required"},
	{Label: "Status Unknown", URL: statusURL, ExpectedStatus: 400, Data: statusUnknown,
		ExpectedResponse: "must have either dlvrd or err set to 1"},
}

func TestHandler(t *testing.T) {
	RunChannelTestCases(t, testChannels, newHandler(), handleTestCases)
}

func BenchmarkHandler(b *testing.B) {
	RunChannelBenchmarks(b, testChannels, newHandler(), handleTestCases)
}

// setSendURL takes care of setting the send_url to our test server host
func setSendURL(s *httptest.Server, h courier.ChannelHandler, c courier.Channel, m courier.Msg) {
	c.(*test.MockChannel).SetConfig("send_url", s.URL)
}

var defaultSendTestCases = []ChannelSendTestCase{
	{Label: "Plain Send",
		MsgText: "Simple Message", MsgURN: "tel:+250788383383",
		ExpectedStatus:   "W",
		MockResponseBody: `Success "External ID1"`, MockResponseStatus: 200, ExpectedExternalID: "External ID1",
		ExpectedURLParams: map[string]string{"content": "Simple Message", "to": "250788383383", "coding": "0",
			"dlr-level": "2", "dlr": "yes", "dlr-method": http.MethodPost, "username": "Username", "password": "Password",
			"dlr-url": "https://localhost/c/js/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/status"},
		SendPrep: setSendURL},
	{Label: "Unicode Send",
		MsgText:          "☺",
		ExpectedStatus:   "W",
		MockResponseBody: `Success "External ID1"`, MockResponseStatus: 200,
		ExpectedURLParams: map[string]string{"content": "?"},
		SendPrep:          setSendURL},
	{Label: "Smart Encoding",
		MsgText: "Fancy “Smart” Quotes", MsgURN: "tel:+250788383383", MsgHighPriority: false,
		ExpectedStatus:   "W",
		MockResponseBody: `Success "External ID1"`, MockResponseStatus: 200,
		ExpectedURLParams: map[string]string{"content": `Fancy "Smart" Quotes`},
		SendPrep:          setSendURL},
	{Label: "Send Attachment",
		MsgText: "My pic!", MsgURN: "tel:+250788383383", MsgHighPriority: true, MsgAttachments: []string{"image/jpeg:https://foo.bar/image.jpg"},
		ExpectedStatus:   "W",
		MockResponseBody: `Success "External ID1"`, MockResponseStatus: 200,
		ExpectedURLParams: map[string]string{"content": "My pic!\nhttps://foo.bar/image.jpg"},
		SendPrep:          setSendURL},
	{Label: "Error Sending",
		MsgText: "Error Message", MsgURN: "tel:+250788383383", MsgHighPriority: false,
		ExpectedStatus:   "E",
		MockResponseBody: "Failed Sending", MockResponseStatus: 401,
		ExpectedURLParams: map[string]string{"content": `Error Message`},
		SendPrep:          setSendURL},
}

func TestSending(t *testing.T) {
	var defaultChannel = test.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "JS", "2020", "US",
		map[string]interface{}{
			"password": "Password",
			"username": "Username"})

	RunChannelSendTestCases(t, defaultChannel, newHandler(), defaultSendTestCases, nil)
}

package blackmyna

import (
	"net/http/httptest"
	"testing"

	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/test"
)

var testChannels = []courier.Channel{
	test.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "BM", "2020", "US", nil),
}

var (
	receiveURL = "/c/bm/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive/"
	statusURL  = "/c/bm/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/status/"

	emptyReceive = receiveURL + ""
	validReceive = receiveURL + "?to=3344&smsc=ncell&from=%2B9779814641111&text=Msg"
	invalidURN   = receiveURL + "?to=3344&smsc=ncell&from=MTN&text=Msg"
	missingText  = receiveURL + "?to=3344&smsc=ncell&from=%2B9779814641111"

	missingStatus = statusURL + "?"
	invalidStatus = statusURL + "?id=bmID&status=13"
	validStatus   = statusURL + "?id=bmID&status=2"
)

var testCases = []ChannelHandleTestCase{
	{Label: "Receive Valid", URL: validReceive, ExpectedStatus: 200, ExpectedResponse: "Message Accepted",
		ExpectedMsgText: Sp("Msg"), ExpectedURN: Sp("tel:+9779814641111")},
	{Label: "Invalid URN", URL: invalidURN, ExpectedStatus: 400, ExpectedResponse: "phone number supplied is not a number"},
	{Label: "Receive Empty", URL: emptyReceive, ExpectedStatus: 400, ExpectedResponse: "field 'text' required"},
	{Label: "Receive Missing Text", URL: missingText, ExpectedStatus: 400, ExpectedResponse: "field 'text' required"},

	{Label: "Status Invalid", URL: invalidStatus, ExpectedStatus: 400, ExpectedResponse: "unknown status"},
	{Label: "Status Missing", URL: missingStatus, ExpectedStatus: 400, ExpectedResponse: "field 'status' required"},
	{Label: "Valid Status", URL: validStatus, ExpectedStatus: 200, ExpectedResponse: `"status":"F"`},
}

func TestHandler(t *testing.T) {
	RunChannelTestCases(t, testChannels, newHandler(), testCases)
}

func BenchmarkHandler(b *testing.B) {
	RunChannelBenchmarks(b, testChannels, newHandler(), testCases)
}

// setSend takes care of setting the sendURL to call
func setSendURL(s *httptest.Server, h courier.ChannelHandler, c courier.Channel, m courier.Msg) {
	sendURL = s.URL
}

var defaultSendTestCases = []ChannelSendTestCase{
	{
		Label:              "Plain Send",
		MsgText:            "Simple Message",
		MsgURN:             "tel:+250788383383",
		MockResponseBody:   `[{"id": "1002"}]`,
		MockResponseStatus: 200,
		ExpectedHeaders:    map[string]string{"Authorization": "Basic VXNlcm5hbWU6UGFzc3dvcmQ="},
		ExpectedPostParams: map[string]string{"message": "Simple Message", "address": "+250788383383", "senderaddress": "2020"},
		ExpectedStatus:     "W",
		ExpectedExternalID: "1002",
		SendPrep:           setSendURL,
	},
	{
		Label:              "Unicode Send",
		MsgText:            "☺",
		MsgURN:             "tel:+250788383383",
		MockResponseBody:   `[{"id": "1002"}]`,
		MockResponseStatus: 200,
		ExpectedPostParams: map[string]string{"message": "☺", "address": "+250788383383", "senderaddress": "2020"},
		ExpectedStatus:     "W",
		ExpectedExternalID: "1002",
		SendPrep:           setSendURL,
	},
	{
		Label:              "Send Attachment",
		MsgText:            "My pic!",
		MsgURN:             "tel:+250788383383",
		MsgAttachments:     []string{"image/jpeg:https://foo.bar/image.jpg"},
		MockResponseBody:   `[{ "id": "1002" }]`,
		MockResponseStatus: 200,
		ExpectedPostParams: map[string]string{"message": "My pic!\nhttps://foo.bar/image.jpg", "address": "+250788383383", "senderaddress": "2020"},
		ExpectedStatus:     "W",
		ExpectedExternalID: "1002",
		SendPrep:           setSendURL,
	},
	{
		Label:              "No External Id",
		MsgText:            "No External ID",
		MsgURN:             "tel:+250788383383",
		MockResponseBody:   `{ "error": "failed" }`,
		MockResponseStatus: 200,
		ExpectedErrors:     []string{"no external id returned in body"},
		ExpectedPostParams: map[string]string{"message": `No External ID`, "address": "+250788383383", "senderaddress": "2020"},
		ExpectedStatus:     "E",
		SendPrep:           setSendURL,
	},
	{
		Label:              "Error Sending",
		MsgText:            "Error Message",
		MsgURN:             "tel:+250788383383",
		MockResponseBody:   `{ "error": "failed" }`,
		MockResponseStatus: 401,
		ExpectedPostParams: map[string]string{"message": `Error Message`, "address": "+250788383383", "senderaddress": "2020"},
		ExpectedStatus:     "E",
		SendPrep:           setSendURL,
	},
}

func TestSending(t *testing.T) {
	var defaultChannel = test.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "BM", "2020", "US",
		map[string]interface{}{
			courier.ConfigPassword: "Password",
			courier.ConfigUsername: "Username",
			courier.ConfigAPIKey:   "KEY",
		})

	RunChannelSendTestCases(t, defaultChannel, newHandler(), defaultSendTestCases, nil)
}

package novo

import (
	"net/http/httptest"
	"testing"

	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/test"
)

var testChannels = []courier.Channel{
	test.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "NV", "2020", "TT", map[string]interface{}{
		"merchant_id":     "my-merchant-id",
		"merchant_secret": "my-merchant-secret",
		"secret":          "sesame",
	}),
}

var (
	receiveURL = "/c/nv/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive/"

	validReceive  = "text=Msg&from=18686846481"
	missingNumber = "text=Msg"
)

var testCases = []ChannelHandleTestCase{
	{Label: "Receive Valid", URL: receiveURL, Data: validReceive, ExpectedStatus: 200, ExpectedResponse: "Message Accepted",
		ExpectedMsgText: Sp("Msg"), ExpectedURN: Sp("tel:+18686846481"), Headers: map[string]string{"Authorization": "sesame"}},
	{Label: "Receive Missing Number", URL: receiveURL, Data: missingNumber, ExpectedStatus: 400, ExpectedResponse: "required field 'from'",
		Headers: map[string]string{"Authorization": "sesame"}},
	{Label: "Receive Missing Authorization", URL: receiveURL, Data: validReceive, ExpectedStatus: 401, ExpectedResponse: "invalid Authorization header",
		ExpectedMsgText: Sp("Msg"), ExpectedURN: Sp("tel:+18686846481")},
}

func TestHandler(t *testing.T) {
	RunChannelTestCases(t, testChannels, newHandler(), testCases)
}

func BenchmarkHandler(b *testing.B) {
	RunChannelBenchmarks(b, testChannels, newHandler(), testCases)
}

func setSendURL(s *httptest.Server, h courier.ChannelHandler, c courier.Channel, m courier.Msg) {
	sendURL = s.URL + "?%s"
}

var defaultSendTestCases = []ChannelSendTestCase{
	{
		Label:              "Plain Send",
		MsgText:            "Simple Message ☺",
		MsgURN:             "tel:+18686846481",
		MockResponseBody:   `{"blastId": "-437733473338","status": "FINISHED","type": "SMS","statusDescription": "Finished"}`,
		MockResponseStatus: 200,
		ExpectedStatus:     "W",
		ExpectedExternalID: "",
		SendPrep:           setSendURL,
	},
	{
		Label:              "Long Send",
		MsgText:            "This is a longer message than 160 characters and will cause us to split it into two separate parts, isn't that right but it is even longer than before I say, I need to keep adding more things to make it work",
		MsgURN:             "tel:+18686846481",
		MockResponseBody:   `{"blastId": "-437733473338","status": "FINISHED","type": "SMS","statusDescription": "Finished"}`,
		MockResponseStatus: 200,
		ExpectedStatus:     "W",
		ExpectedExternalID: "",
		SendPrep:           setSendURL,
	},
	{
		Label:              "Send Attachment",
		MsgText:            "My pic!",
		MsgURN:             "tel:+18686846481",
		MsgAttachments:     []string{"image/jpeg:https://foo.bar/image.jpg"},
		MockResponseBody:   `{"blastId": "-437733473338","status": "FINISHED","type": "SMS","statusDescription": "Finished"}`,
		MockResponseStatus: 200,
		ExpectedStatus:     "W",
		ExpectedExternalID: "",
		SendPrep:           setSendURL,
	},
	{
		Label:              "Invalid Parameters",
		MsgText:            "Invalid Parameters",
		MsgURN:             "tel:+18686846481",
		MockResponseBody:   `{"error": "Incorrect Query String Authentication ","expectedQueryString": "8868;18686846480;test;"}`,
		MockResponseStatus: 200,
		ExpectedStatus:     "F",
		ExpectedErrors:     []string{"received invalid response"},
		SendPrep:           setSendURL,
	},
	{
		Label:              "Error Response",
		MsgText:            "Error Response",
		MsgURN:             "tel:+18686846481",
		MockResponseBody:   `{"error": "Incorrect Query String Authentication ","expectedQueryString": "8868;18686846480;test;"}`,
		MockResponseStatus: 200,
		ExpectedStatus:     "F",
		ExpectedErrors:     []string{"received invalid response"},
		SendPrep:           setSendURL,
	},
}

func TestSending(t *testing.T) {
	maxMsgLength = 160
	var defaultChannel = test.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "NV", "2020", "TT",
		map[string]interface{}{
			"merchant_id":     "my-merchant-id",
			"merchant_secret": "my-merchant-secret",
			"secret":          "sesame",
		})
	RunChannelSendTestCases(t, defaultChannel, newHandler(), defaultSendTestCases, nil)
}

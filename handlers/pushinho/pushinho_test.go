package pushinho

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"github.com/sirupsen/logrus"
)

var testChannels = []courier.Channel{
	courier.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c568c", "PS", "1234", "", map[string]interface{}{}),
}

func setSendURL(s *httptest.Server, h courier.ChannelHandler, c courier.Channel, m courier.Msg) {
	sendURL = s.URL
}

var (
	// TODO: change the receive url to the correct uuid
	receiveURL  = "/c/ps/8eb23e93-5ecb-45ba-b726-3b064e0c568c/receive"
	validMsg    = "from=yl-UYhnSFDNYvGDqAVOK&text=hello+world"
	missingText = "from=yl-UYhnSFDNYvGDqAVOK"
	missingFrom = "text=hello+world"
)

var sendTestCases = []ChannelSendTestCase{
	{Label: "Plain Send",
		Text:           "Simple Message",
		URN:            "fcm:yl-UYhnSFDNYvGDqAVOK",
		Status:         "W",
		ResponseBody:   "success",
		ResponseStatus: 200,
		// TODO: change the fcm: from the RequestBody
		RequestBody: `{"to":"fcm:yl-UYhnSFDNYvGDqAVOK","text":"Simple Message","metadata":{}}`,
		SendPrep:    setSendURL},
	{Label: "With Quick Replies",
		Text:           "Simple Message",
		URN:            "fcm:yl-UYhnSFDNYvGDqAVOK",
		Status:         "W",
		ResponseBody:   "success",
		ResponseStatus: 200,
		//Metadata: []byte(`
		//	"quick_replies": [
		//		{
		//			"title": "First button"
		//		},
		//		{
		//			"title": "Second button"
		//		}
		//	]
		//`),
		QuickReplies: []string{"First button", "Second button"},
		// TODO: change the fcm: from the RequestBody
		RequestBody: `{"to":"fcm:yl-UYhnSFDNYvGDqAVOK","text":"Simple Message","metadata":{"quick_replies":[{"title":"First button"},{"title":"Second button"}]}}`,
		SendPrep:    setSendURL,
	},
}

var receiveTestCase = []ChannelHandleTestCase{
	{Label: "Receive Valid Message", URL: receiveURL, Data: validMsg, Status: 200, Response: "Accepted",
		Text: Sp("hello world"), URN: Sp("fcm:yl-UYhnSFDNYvGDqAVOK")},
	{Label: "Receive Missing From", URL: receiveURL, Data: missingFrom, Status: 400, Response: "field 'from' required"},
	{Label: "Receive Missing Text", URL: receiveURL, Data: missingText, Status: 200, Response: "Accepted"},
}

func newServer(backend courier.Backend) courier.Server {
	// for benchmarks, log to null
	logger := logrus.New()
	logger.Out = ioutil.Discard
	logrus.SetOutput(ioutil.Discard)

	return courier.NewServerWithLogger(courier.NewConfig(), backend, logger)
}

func TestReceiveMessage(t *testing.T) {
	RunChannelTestCases(t, testChannels, newHandler(), receiveTestCase)
}

func TestSendMessage(t *testing.T) {
	RunChannelSendTestCases(t, testChannels[0], newHandler(), sendTestCases, nil)
}

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
	// courier.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c568c", "PS", "1234", "", map[string]interface{}{}),
	courier.NewMockChannel("781ea439-470d-4d9f-8045-115ec4d71001", "PS", "1234", "", map[string]interface{}{}),
}

func setSendURL(s *httptest.Server, h courier.ChannelHandler, c courier.Channel, m courier.Msg) {
	sendURL = s.URL
}

var (
	// TODO: change the receive url to the correct uuid
	receiveURL  = "/c/ps/781ea439-470d-4d9f-8045-115ec4d71001/receive"
	validMsg    = "from=yl-UYhnSFDNYvGDqAVOK&text=hello+world"
	missingText = "from=yl-UYhnSFDNYvGDqAVOK"
	missingFrom = "text=hello+world"
	task        = "from=yl-UYhnSFDNYvGDqAVOK&text=asd"
	taskURL     = "/c/ps/781ea439-470d-4d9f-8045-115ec4d71001/receive"
)

var sendTestCases = []ChannelSendTestCase{
	{Label: "Plain Send",
		Text:           "Simple Message",
		URN:            "ps:yl-UYhnSFDNYvGDqAVOK",
		Status:         "W",
		ResponseBody:   "success",
		ResponseStatus: 200,
		// TODO: change the fcm: from the RequestBody
		RequestBody: `{"to":"ps:yl-UYhnSFDNYvGDqAVOK","text":"Simple Message","metadata":{}}`,
		SendPrep:    setSendURL},
	{Label: "With Quick Replies",
		Text:           "Simple Message",
		URN:            "ps:yl-UYhnSFDNYvGDqAVOK",
		Status:         "W",
		ResponseBody:   "success",
		ResponseStatus: 200,
		Metadata: []byte(`
			{
				"quick_replies": [
					{
						"title": "First button"
					},
					{
						"title": "Second button"
					}
				]
			}
		`),
		// TODO: change the fcm: from the RequestBody
		RequestBody: `{"to":"ps:yl-UYhnSFDNYvGDqAVOK","text":"Simple Message","metadata":{"quick_replies":[{"title":"First button"},{"title":"Second button"}]}}`,
		SendPrep:    setSendURL,
	},
}

var receiveTestCase = []ChannelHandleTestCase{
	{Label: "Receive Valid Message", URL: receiveURL, Data: validMsg, Status: 200, Response: "Accepted",
		Text: Sp("hello world"), URN: Sp("ps:yl-UYhnSFDNYvGDqAVOK")},
	{Label: "Receive Missing From", URL: receiveURL, Data: missingFrom, Status: 400, Response: "field 'from' required"},
	{Label: "Receive Missing Text", URL: receiveURL, Data: missingText, Status: 200, Response: "Accepted"},
	{Label: "Task", URL: taskURL, Data: task, Status: 200, Response: "Accepted"},
}

func newServer(backend courier.Backend) courier.Server {
	// for benchmarks, log to null
	logger := logrus.New()
	logger.Out = ioutil.Discard
	logrus.SetOutput(ioutil.Discard)

	return courier.NewServerWithLogger(courier.NewConfig(), backend, logger)
}

func MyFunction() {

}

func TestReceiveMessage(t *testing.T) {
	RunChannelTestCases(t, testChannels, newHandler(), receiveTestCase)
	MyFunction()
}

func TestSendMessage(t *testing.T) {
	RunChannelSendTestCases(t, testChannels[0], newHandler(), sendTestCases, nil)
}

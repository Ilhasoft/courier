package discord

import (
	"net/http/httptest"
	"testing"

	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/test"
	"github.com/nyaruka/courier/utils"
)

func TestHandler(t *testing.T) {
	RunChannelTestCases(t, testChannels, newHandler(), testCases)
}

func BenchmarkHandler(b *testing.B) {
	RunChannelBenchmarks(b, testChannels, newHandler(), testCases)
}

var testChannels = []courier.Channel{
	test.NewMockChannel("bac782c2-7aeb-4389-92f5-97887744f573", "DS", "discord", "US", map[string]interface{}{}),
}

var testCases = []ChannelHandleTestCase{
	{Label: "Recieve Message", URL: "/c/ds/bac782c2-7aeb-4389-92f5-97887744f573/receive", Data: `from=694634743521607802&text=hello`, ExpectedStatus: 200, ExpectedMsgText: Sp("hello"), ExpectedURN: Sp("discord:694634743521607802")},
	{Label: "Recieve Message with attachment", URL: "/c/ds/bac782c2-7aeb-4389-92f5-97887744f573/receive", Data: `from=694634743521607802&text=hello&attachments=https://test.test/foo.png`, ExpectedStatus: 200, ExpectedMsgText: Sp("hello"), ExpectedURN: Sp("discord:694634743521607802"), ExpectedAttachments: []string{"https://test.test/foo.png"}},
	{Label: "Invalid ID", URL: "/c/ds/bac782c2-7aeb-4389-92f5-97887744f573/receive", Data: `from=somebody&text=hello`, ExpectedStatus: 400, ExpectedResponse: "Error"},
	{Label: "Garbage Body", URL: "/c/ds/bac782c2-7aeb-4389-92f5-97887744f573/receive", Data: `sdfaskdfajsdkfajsdfaksdf`, ExpectedStatus: 400, ExpectedResponse: "Error"},
	{Label: "Missing Text", URL: "/c/ds/bac782c2-7aeb-4389-92f5-97887744f573/receive", Data: `from=694634743521607802`, ExpectedStatus: 400, ExpectedResponse: "Error"},
	{Label: "Message Sent Handler", URL: "/c/ds/bac782c2-7aeb-4389-92f5-97887744f573/sent/", Data: `id=12345`, ExpectedStatus: 200, ExpectedResponse: `"status":"S"`},
	{Label: "Message Sent Handler Garbage", URL: "/c/ds/bac782c2-7aeb-4389-92f5-97887744f573/sent/", Data: `nothing`, ExpectedStatus: 400},
}

var sendTestCases = []ChannelSendTestCase{
	{Label: "Simple Send", MsgText: "Hello World", MsgURN: "discord:694634743521607802", ExpectedRequestPath: "/discord/rp/send", MockResponseStatus: 200, ExpectedRequestBody: `{"id":"10","text":"Hello World","to":"694634743521607802","channel":"bac782c2-7aeb-4389-92f5-97887744f573","attachments":[],"quick_replies":null}`, SendPrep: setSendURL},
	{Label: "Simple Send", MsgText: "Hello World", MsgAttachments: []string{"image/jpeg:https://foo.bar/image.jpg"}, MsgURN: "discord:694634743521607802", ExpectedRequestPath: "/discord/rp/send", ExpectedRequestBody: `{"id":"10","text":"Hello World","to":"694634743521607802","channel":"bac782c2-7aeb-4389-92f5-97887744f573","attachments":["https://foo.bar/image.jpg"],"quick_replies":null}`, MockResponseStatus: 200, SendPrep: setSendURL},
	{Label: "Simple Send with attachements and Quick Replies", MsgText: "Hello World", MsgAttachments: []string{"image/jpeg:https://foo.bar/image.jpg"}, MsgQuickReplies: []string{"hello", "world"}, MsgURN: "discord:694634743521607802", ExpectedRequestPath: "/discord/rp/send", ExpectedRequestBody: `{"id":"10","text":"Hello World","to":"694634743521607802","channel":"bac782c2-7aeb-4389-92f5-97887744f573","attachments":["https://foo.bar/image.jpg"],"quick_replies":["hello","world"]}`, MockResponseStatus: 200, SendPrep: setSendURL},
}

// setSendURL takes care of setting the send_url to our test server host
func setSendURL(s *httptest.Server, h courier.ChannelHandler, c courier.Channel, m courier.Msg) {
	// this is actually a path, which we'll combine with the test server URL
	sendURL := c.StringConfigForKey("send_path", "/discord/rp/send")
	sendURL, _ = utils.AddURLPath(s.URL, sendURL)
	c.(*test.MockChannel).SetConfig(courier.ConfigSendURL, sendURL)
}
func TestSending(t *testing.T) {

	RunChannelSendTestCases(t, testChannels[0], newHandler(), sendTestCases, nil)
}

package kaleyra

import (
	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	channelUUID      = "8eb23e93-5ecb-45ba-b726-3b064e0c568c"
	receiveMsgURL    = "/c/kwa/" + channelUUID + "/receive"
	receiveStatusURL = "/c/kwa/" + channelUUID + "/status"
)

var testChannels = []courier.Channel{
	courier.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c568c", "KWA", "250788383383", "",
		map[string]interface{}{
			configAccountSID: "SID",
			configApiKey:     "123456",
		},
	),
}

var testCases = []ChannelHandleTestCase{
	{
		Label:       "Receive Msg",
		URL:         receiveMsgURL + "?created_at=1603914166&type=text&from=14133881111&name=John%20Cruz&body=Hello%20World",
		Name:        Sp("John Cruz"),
		URN:         Sp("whatsapp:14133881111"),
		Text:        Sp("Hello World"),
		Attachments: []string{},
		Status:      200,
		Response:    "Accepted",
	},
	{
		Label:      "Receive Media",
		URL:        receiveMsgURL + "?created_at=1603914166&type=image&from=14133881111&name=John%20Cruz&media_url=https://link.to/image.jpg",
		Name:       Sp("John Cruz"),
		URN:        Sp("whatsapp:14133881111"),
		Text:       Sp(""),
		Attachment: Sp("https://link.to/image.jpg"),
		Status:     200,
		Response:   "Accepted",
	},
	{
		Label:    "Receive Empty",
		URL:      receiveMsgURL + "?created_at=1603914166&type=text&from=14133881111&name=John%20Cruz",
		Status:   400,
		Response: "no text or media",
	},
	{
		Label:    "Receive Invalid CreatedAt",
		URL:      receiveMsgURL + "?created_at=nottimestamp&type=text&from=14133881111&name=John%20Cruz&body=Hi",
		Name:     Sp("John Cruz"),
		Status:   400,
		Response: "invalid created_at",
	},
	{
		Label:    "Receive Invalid Type",
		URL:      receiveMsgURL + "?created_at=1603914166&type=sticker&from=14133881111&name=John%20Cruz",
		Status:   200,
		Response: "unknown message type",
	},
	{
		Label:    "Receive Invalid From",
		URL:      receiveMsgURL + "?created_at=1603914166&type=text&from=notnumber&name=John%20Cruz&body=Hi",
		Name:     Sp("John Cruz"),
		Status:   400,
		Response: "invalid whatsapp id",
	},

	{
		Label:      "Receive Valid Status",
		URL:        receiveStatusURL + "?id=58f86fab-85c5-4f7c-9b68-9c323248afc4%3A0&status=read",
		ExternalID: Sp("58f86fab-85c5-4f7c-9b68-9c323248afc4:0"),
		MsgStatus:  Sp("D"),
		Status:     200,
		Response:   `"type":"status"`,
	},
	{
		Label:      "Receive Invalid Status",
		URL:        receiveStatusURL + "?id=58f86fab-85c5-4f7c-9b68-9c323248afc4%3A0&status=deleted",
		ExternalID: Sp("58f86fab-85c5-4f7c-9b68-9c323248afc4:0"),
		MsgStatus:  Sp("D"),
		Status:     200,
		Response:   "unknown status",
	},
}

func TestHandler(t *testing.T) {
	RunChannelTestCases(t, testChannels, newHandler(), testCases)
}

func BenchmarkHandler(b *testing.B) {
	RunChannelBenchmarks(b, testChannels, newHandler(), testCases)
}

func setSendURL(s *httptest.Server, h courier.ChannelHandler, c courier.Channel, m courier.Msg) {
	baseURL = s.URL
}

var sendTestCases = []ChannelSendTestCase{
	{
		Label:          "Plain Send",
		Text:           "Simple Message",
		URN:            "whatsapp:14133881111",
		Status:         "W",
		ExternalID:     "58f86fab-85c5-4f7c-9b68-9c323248afc4:0",
		Path:           "/v1/SID/messages",
		Headers:        map[string]string{"Content-type": "application/x-www-form-urlencoded"},
		RequestBody:    "api-key=123456&body=Simple+Message&callback_url=https%3A%2F%2Flocalhost%2Fc%2Fkwa%2F8eb23e93-5ecb-45ba-b726-3b064e0c568c%2Fstatus&channel=WhatsApp&from=250788383383&to=14133881111&type=text",
		ResponseStatus: 200,
		ResponseBody:   `{"id":"58f86fab-85c5-4f7c-9b68-9c323248afc4:0"}`,
		SendPrep:       setSendURL,
	},
	{
		Label:          "Unicode Send",
		Text:           "☺",
		URN:            "whatsapp:14133881111",
		Status:         "W",
		ExternalID:     "58f86fab-85c5-4f7c-9b68-9c323248afc4:0",
		Path:           "/v1/SID/messages",
		Headers:        map[string]string{"Content-type": "application/x-www-form-urlencoded"},
		RequestBody:    "api-key=123456&body=%E2%98%BA&callback_url=https%3A%2F%2Flocalhost%2Fc%2Fkwa%2F8eb23e93-5ecb-45ba-b726-3b064e0c568c%2Fstatus&channel=WhatsApp&from=250788383383&to=14133881111&type=text",
		ResponseStatus: 200,
		ResponseBody:   `{"id":"58f86fab-85c5-4f7c-9b68-9c323248afc4:0"}`,
		SendPrep:       setSendURL,
	},
	{
		Label:          "Plain Send Error",
		Text:           "Error",
		URN:            "whatsapp:14133881112",
		Status:         "F",
		Path:           "/v1/SID/messages",
		Headers:        map[string]string{"Content-type": "application/x-www-form-urlencoded"},
		RequestBody:    "api-key=123456&body=Error&callback_url=https%3A%2F%2Flocalhost%2Fc%2Fkwa%2F8eb23e93-5ecb-45ba-b726-3b064e0c568c%2Fstatus&channel=WhatsApp&from=250788383383&to=14133881112&type=text",
		ResponseStatus: 400,
		ResponseBody:   `{"error":{"to":"invalid number"}}`,
		SendPrep:       setSendURL,
	},
	{
		Label:       "Medias Send",
		Text:        "Medias",
		Attachments: []string{"image/jpg:https://foo.bar/image.jpg", "image/png:https://foo.bar/video.mp4"},
		URN:         "whatsapp:14133881111",
		Status:      "W",
		ExternalID:  "f75fbe1e-a0c0-4923-96e8-5043aa617b2b:0",
		Responses: map[MockedRequest]MockedResponse{
			MockedRequest{
				Method:       "POST",
				Path:         "/v1/SID/messages",
				BodyContains: "image bytes",
			}: {
				Status: 200,
				Body:   `{"id":"58f86fab-85c5-4f7c-9b68-9c323248afc4:0"}`,
			},
			MockedRequest{
				Method:       "POST",
				Path:         "/v1/SID/messages",
				BodyContains: "video bytes",
			}: {
				Status: 200,
				Body:   `{"id":"f75fbe1e-a0c0-4923-96e8-5043aa617b2b:0"}`,
			},
		},
		SendPrep: setSendURL,
	},
	{
		Label:       "Media Send Error",
		Text:        "Medias",
		Attachments: []string{"image/jpg:https://foo.bar/image.jpg", "image/png:https://foo.bar/video.wmv"},
		URN:         "whatsapp:14133881111",
		Status:      "F",
		Responses: map[MockedRequest]MockedResponse{
			MockedRequest{
				Method:       "POST",
				Path:         "/v1/SID/messages",
				BodyContains: "image bytes",
			}: {
				Status: 200,
				Body:   `{"id":"58f86fab-85c5-4f7c-9b68-9c323248afc4:0"}`,
			},
			MockedRequest{
				Method:       "POST",
				Path:         "/v1/SID/messages",
				BodyContains: "video bytes",
			}: {
				Status: 400,
				Body:   `{"error":{"media":"invalid media type"}}`,
			},
		},
		SendPrep: setSendURL,
	},
}

func mockAttachmentURLs(mediaServer *httptest.Server, testCases []ChannelSendTestCase) []ChannelSendTestCase {
	casesWithMockedUrls := make([]ChannelSendTestCase, len(testCases))

	for i, testCase := range testCases {
		mockedCase := testCase

		for j, attachment := range testCase.Attachments {
			mockedCase.Attachments[j] = strings.Replace(attachment, "https://foo.bar", mediaServer.URL, 1)
		}
		casesWithMockedUrls[i] = mockedCase
	}
	return casesWithMockedUrls
}

func TestSending(t *testing.T) {
	mediaServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		res.WriteHeader(200)

		path := req.URL.Path
		if strings.Contains(path, "image") {
			res.Write([]byte("image bytes"))
		} else if strings.Contains(path, "video") {
			res.Write([]byte("video bytes"))
		} else {
			res.Write([]byte("media bytes"))
		}
	}))
	mockedSendTestCases := mockAttachmentURLs(mediaServer, sendTestCases)

	RunChannelSendTestCases(t, testChannels[0], newHandler(), mockedSendTestCases, nil)
}

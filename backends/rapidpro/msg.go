package rapidpro

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gomodule/redigo/redis"
	"github.com/lib/pq"
	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/queue"
	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/null"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	filetype "gopkg.in/h2non/filetype.v1"
)

// MsgDirection is the direction of a message
type MsgDirection string

// Possible values for MsgDirection
const (
	MsgIncoming     MsgDirection = "I"
	MsgOutgoing     MsgDirection = "O"
	NilMsgDirection MsgDirection = ""
)

// MsgVisibility is the visibility of a message
type MsgVisibility string

// Possible values for MsgVisibility
const (
	MsgVisible  MsgVisibility = "V"
	MsgDeleted  MsgVisibility = "D"
	MsgArchived MsgVisibility = "A"
)

// WriteMsg creates a message given the passed in arguments
func writeMsg(ctx context.Context, b *backend, msg courier.Msg, clog *courier.ChannelLog) error {
	m := msg.(*DBMsg)

	// this msg has already been written (we received it twice), we are a no op
	if m.alreadyWritten {
		return nil
	}

	channel := m.Channel()

	// check for data: attachment URLs which need to be fetched now - fetching of other URLs can be deferred until
	// message handling and performed by calling the /c/_fetch-attachment endpoint
	for i, attURL := range m.Attachments_ {
		if strings.HasPrefix(attURL, "data:") {
			attData, err := base64.StdEncoding.DecodeString(attURL[5:])
			if err != nil {
				clog.Error(courier.ErrorAttachmentNotDecodable())
				return errors.Wrap(err, "unable to decode attachment data")
			}

			var contentType, extension string
			fileType, _ := filetype.Match(attData[:300])
			if fileType != filetype.Unknown {
				contentType = fileType.MIME.Value
				extension = fileType.Extension
			} else {
				contentType = "application/octet-stream"
				extension = "bin"
			}

			newURL, err := b.SaveAttachment(ctx, channel, contentType, attData, extension)
			if err != nil {
				return err
			}
			m.Attachments_[i] = fmt.Sprintf("%s:%s", contentType, newURL)
		}
	}

	// try to write it our db
	err := writeMsgToDB(ctx, b, m, clog)

	// fail? log
	if err != nil {
		logrus.WithError(err).WithField("msg", m.UUID().String()).Error("error writing to db")
	}

	// if we failed write to spool
	if err != nil {
		err = courier.WriteToSpool(b.config.SpoolDir, "msgs", m)
	}
	// mark this msg as having been seen
	b.writeMsgSeen(m)
	return err
}

// newMsg creates a new DBMsg object with the passed in parameters
func newMsg(direction MsgDirection, channel courier.Channel, urn urns.URN, text string, clog *courier.ChannelLog) *DBMsg {
	now := time.Now()
	dbChannel := channel.(*DBChannel)

	return &DBMsg{
		OrgID_:        dbChannel.OrgID(),
		UUID_:         courier.NewMsgUUID(),
		Direction_:    direction,
		Status_:       courier.MsgPending,
		Visibility_:   MsgVisible,
		HighPriority_: false,
		Text_:         text,

		ChannelID_:   dbChannel.ID(),
		ChannelUUID_: dbChannel.UUID(),

		URN_:          urn,
		MessageCount_: 1,

		NextAttempt_: now,
		CreatedOn_:   now,
		ModifiedOn_:  now,
		QueuedOn_:    now,
		LogUUIDs:     []string{string(clog.UUID())},

		channel:        dbChannel,
		workerToken:    "",
		alreadyWritten: false,
	}
}

const sqlInsertMsg = `
INSERT INTO
	msgs_msg(org_id, uuid, direction, text, attachments, msg_count, error_count, high_priority, status,
             visibility, external_id, channel_id, contact_id, contact_urn_id, created_on, modified_on, next_attempt, queued_on, sent_on, log_uuids)
    VALUES(:org_id, :uuid, :direction, :text, :attachments, :msg_count, :error_count, :high_priority, :status,
           :visibility, :external_id, :channel_id, :contact_id, :contact_urn_id, :created_on, :modified_on, :next_attempt, :queued_on, :sent_on, :log_uuids)
RETURNING id`

func writeMsgToDB(ctx context.Context, b *backend, m *DBMsg, clog *courier.ChannelLog) error {
	// grab the contact for this msg
	contact, err := contactForURN(ctx, b, m.OrgID_, m.channel, m.URN_, m.URNAuth_, m.ContactName_, clog)

	// our db is down, write to the spool, we will write/queue this later
	if err != nil {
		return errors.Wrap(err, "error getting contact for message")
	}

	// set our contact and urn id
	m.ContactID_ = contact.ID_
	m.ContactURNID_ = contact.URNID_

	rows, err := b.db.NamedQueryContext(ctx, sqlInsertMsg, m)
	if err != nil {
		return errors.Wrap(err, "error inserting message")
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&m.ID_)
	if err != nil {
		return errors.Wrap(err, "error scanning for inserted message id")
	}

	// queue this up to be handled by RapidPro
	rc := b.redisPool.Get()
	defer rc.Close()
	err = queueMsgHandling(rc, contact, m)

	// if we had a problem queueing the handling, log it, but our message is written, it'll
	// get picked up by our rapidpro catch-all after a period
	if err != nil {
		logrus.WithError(err).WithField("msg_id", m.ID_).Error("error queueing msg handling")
	}

	return nil
}

const sqlSelectMsg = `
SELECT
	org_id,
	direction,
	text,
	attachments,
	msg_count,
	error_count,
	failed_reason,
	high_priority,
	status,
	visibility,
	external_id,
	channel_id,
	contact_id,
	contact_urn_id,
	created_on,
	modified_on,
	next_attempt,
	queued_on,
	sent_on,
	log_uuids
FROM
	msgs_msg
WHERE
	id = $1`

const selectChannelSQL = `
SELECT
	org_id,
	ch.id as id,
	ch.uuid as uuid,
	ch.name as name,
	channel_type, schemes,
	address, role,
	ch.country as country,
	ch.config as config,
	org.config as org_config,
	org.is_anon as org_is_anon
FROM
	channels_channel ch
	JOIN orgs_org org on ch.org_id = org.id
WHERE
    ch.id = $1
`

//-----------------------------------------------------------------------------
// Msg flusher for flushing failed writes
//-----------------------------------------------------------------------------

func (b *backend) flushMsgFile(filename string, contents []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	msg := &DBMsg{}
	err := json.Unmarshal(contents, msg)
	if err != nil {
		log.Printf("ERROR unmarshalling spool file '%s', renaming: %s\n", filename, err)
		os.Rename(filename, fmt.Sprintf("%s.error", filename))
		return nil
	}

	// look up our channel
	channel, err := b.GetChannel(ctx, courier.AnyChannelType, msg.ChannelUUID_)
	if err != nil {
		return err
	}
	msg.channel = channel.(*DBChannel)

	// create log tho it won't be written
	clog := courier.NewChannelLog(courier.ChannelLogTypeMsgReceive, channel, nil)

	// try to write it our db
	err = writeMsgToDB(ctx, b, msg, clog)

	// fail? oh well, we'll try again later
	return err
}

//-----------------------------------------------------------------------------
// Deduping utility methods
//-----------------------------------------------------------------------------

// checkMsgSeen tries to look up whether a msg with the fingerprint passed in was seen in window or prevWindow. If
// found returns the UUID of that msg, if not returns empty string
func (b *backend) checkMsgSeen(msg *DBMsg) courier.MsgUUID {
	rc := b.redisPool.Get()
	defer rc.Close()

	uuidAndText, _ := b.seenMsgs.Get(rc, msg.fingerprint(false))

	// if so, test whether the text it the same
	if uuidAndText != "" {
		prevText := uuidAndText[37:]

		// if it is the same, return the UUID
		if prevText == msg.Text() {
			return courier.NewMsgUUIDFromString(uuidAndText[:36])
		}
	}
	return courier.NilMsgUUID
}

// writeMsgSeen writes that the message with the passed in fingerprint and UUID was seen in the
// passed in window
func (b *backend) writeMsgSeen(msg *DBMsg) {
	rc := b.redisPool.Get()
	defer rc.Close()

	b.seenMsgs.Set(rc, msg.fingerprint(false), fmt.Sprintf("%s|%s", msg.UUID().String(), msg.Text()))
}

// clearMsgSeen clears our seen incoming messages for the passed in channel and URN
func (b *backend) clearMsgSeen(rc redis.Conn, msg *DBMsg) {
	b.seenMsgs.Remove(rc, msg.fingerprint(false))
}

func (b *backend) checkExternalIDSeen(msg *DBMsg) courier.MsgUUID {
	rc := b.redisPool.Get()
	defer rc.Close()

	uuidAndText, _ := b.seenExternalIDs.Get(rc, msg.fingerprint(true))

	// if so, test whether the text it the same
	if uuidAndText != "" {
		prevText := uuidAndText[37:]

		// if it is the same, return the UUID
		if prevText == msg.Text() {
			return courier.NewMsgUUIDFromString(uuidAndText[:36])
		}
	}
	return courier.NilMsgUUID
}

func (b *backend) writeExternalIDSeen(msg *DBMsg) {
	rc := b.redisPool.Get()
	defer rc.Close()

	b.seenExternalIDs.Set(rc, msg.fingerprint(true), fmt.Sprintf("%s|%s", msg.UUID().String(), msg.Text()))
}

//-----------------------------------------------------------------------------
// Our implementation of Msg interface
//-----------------------------------------------------------------------------

// DBMsg is our base struct to represent msgs both in our JSON and db representations
type DBMsg struct {
	OrgID_                OrgID                  `json:"org_id"          db:"org_id"`
	ID_                   courier.MsgID          `json:"id"              db:"id"`
	UUID_                 courier.MsgUUID        `json:"uuid"            db:"uuid"`
	Direction_            MsgDirection           `json:"direction"       db:"direction"`
	Status_               courier.MsgStatusValue `json:"status"          db:"status"`
	Visibility_           MsgVisibility          `json:"visibility"      db:"visibility"`
	HighPriority_         bool                   `json:"high_priority"   db:"high_priority"`
	URN_                  urns.URN               `json:"urn"`
	URNAuth_              string                 `json:"urn_auth"`
	Text_                 string                 `json:"text"            db:"text"`
	Attachments_          pq.StringArray         `json:"attachments"     db:"attachments"`
	ExternalID_           null.String            `json:"external_id"     db:"external_id"`
	ResponseToExternalID_ string                 `json:"response_to_external_id"`
	IsResend_             bool                   `json:"is_resend,omitempty"`
	Metadata_             json.RawMessage        `json:"metadata"        db:"metadata"`

	ChannelID_    courier.ChannelID `json:"channel_id"      db:"channel_id"`
	ContactID_    ContactID         `json:"contact_id"      db:"contact_id"`
	ContactURNID_ ContactURNID      `json:"contact_urn_id"  db:"contact_urn_id"`

	MessageCount_ int         `json:"msg_count"     db:"msg_count"`
	ErrorCount_   int         `json:"error_count"   db:"error_count"`
	FailedReason_ null.String `json:"failed_reason" db:"failed_reason"`

	ChannelUUID_ courier.ChannelUUID `json:"channel_uuid"`
	ContactName_ string              `json:"contact_name"`

	NextAttempt_ time.Time      `json:"next_attempt"  db:"next_attempt"`
	CreatedOn_   time.Time      `json:"created_on"    db:"created_on"`
	ModifiedOn_  time.Time      `json:"modified_on"   db:"modified_on"`
	QueuedOn_    time.Time      `json:"queued_on"     db:"queued_on"`
	SentOn_      *time.Time     `json:"sent_on"       db:"sent_on"`
	LogUUIDs     pq.StringArray `json:"log_uuids"     db:"log_uuids"`

	// fields used to allow courier to update a session's timeout when a message is sent for efficient timeout behavior
	SessionID_            SessionID  `json:"session_id,omitempty"`
	SessionTimeout_       int        `json:"session_timeout,omitempty"`
	SessionWaitStartedOn_ *time.Time `json:"session_wait_started_on,omitempty"`
	SessionStatus_        string     `json:"session_status,omitempty"`

	Flow_ *courier.FlowReference `json:"flow,omitempty"`

	channel        *DBChannel
	workerToken    queue.WorkerToken
	alreadyWritten bool
	quickReplies   []string
	locale         string
}

func (m *DBMsg) ID() courier.MsgID            { return m.ID_ }
func (m *DBMsg) EventID() int64               { return int64(m.ID_) }
func (m *DBMsg) UUID() courier.MsgUUID        { return m.UUID_ }
func (m *DBMsg) Text() string                 { return m.Text_ }
func (m *DBMsg) Attachments() []string        { return []string(m.Attachments_) }
func (m *DBMsg) ExternalID() string           { return string(m.ExternalID_) }
func (m *DBMsg) URN() urns.URN                { return m.URN_ }
func (m *DBMsg) URNAuth() string              { return m.URNAuth_ }
func (m *DBMsg) ContactName() string          { return m.ContactName_ }
func (m *DBMsg) HighPriority() bool           { return m.HighPriority_ }
func (m *DBMsg) ReceivedOn() *time.Time       { return m.SentOn_ }
func (m *DBMsg) SentOn() *time.Time           { return m.SentOn_ }
func (m *DBMsg) ResponseToExternalID() string { return m.ResponseToExternalID_ }
func (m *DBMsg) IsResend() bool               { return m.IsResend_ }

func (m *DBMsg) Channel() courier.Channel { return m.channel }
func (m *DBMsg) SessionStatus() string    { return m.SessionStatus_ }

func (m *DBMsg) Flow() *courier.FlowReference { return m.Flow_ }

func (m *DBMsg) FlowName() string {
	if m.Flow_ == nil {
		return ""
	}
	return m.Flow_.Name
}

func (m *DBMsg) FlowUUID() string {
	if m.Flow_ == nil {
		return ""
	}
	return m.Flow_.UUID
}

func (m *DBMsg) QuickReplies() []string {
	if m.quickReplies != nil {
		return m.quickReplies
	}

	if m.Metadata_ == nil {
		return nil
	}

	m.quickReplies = []string{}
	jsonparser.ArrayEach(
		m.Metadata_,
		func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			m.quickReplies = append(m.quickReplies, string(value))
		},
		"quick_replies")
	return m.quickReplies
}

func (m *DBMsg) Topic() string {
	if m.Metadata_ == nil {
		return ""
	}
	topic, _, _, _ := jsonparser.Get(m.Metadata_, "topic")
	return string(topic)
}

func (m *DBMsg) Locale() string {
	if m.locale != "" {
		return m.locale
	}
	if m.Metadata_ == nil {
		return ""
	}

	locale, _, _, _ := jsonparser.Get(m.Metadata_, "locale")
	return string(locale)
}

// Metadata returns the metadata for this message
func (m *DBMsg) Metadata() json.RawMessage {
	return m.Metadata_
}

// fingerprint returns a fingerprint for this msg, suitable for figuring out if this is a dupe
func (m *DBMsg) fingerprint(withExtID bool) string {
	if withExtID {
		return fmt.Sprintf("%s:%s|%s", m.Channel().UUID(), m.URN().Identity(), m.ExternalID())
	}
	return fmt.Sprintf("%s:%s", m.ChannelUUID_, m.URN_.Identity())
}

// WithContactName can be used to set the contact name on a msg
func (m *DBMsg) WithContactName(name string) courier.Msg { m.ContactName_ = name; return m }

// WithReceivedOn can be used to set sent_on on a msg in a chained call
func (m *DBMsg) WithReceivedOn(date time.Time) courier.Msg { m.SentOn_ = &date; return m }

// WithExternalID can be used to set the external id on a msg in a chained call
func (m *DBMsg) WithExternalID(id string) courier.Msg { m.ExternalID_ = null.String(id); return m }

// WithID can be used to set the id on a msg in a chained call
func (m *DBMsg) WithID(id courier.MsgID) courier.Msg { m.ID_ = id; return m }

// WithUUID can be used to set the id on a msg in a chained call
func (m *DBMsg) WithUUID(uuid courier.MsgUUID) courier.Msg { m.UUID_ = uuid; return m }

// WithMetadata can be used to add metadata to a Msg
func (m *DBMsg) WithMetadata(metadata json.RawMessage) courier.Msg { m.Metadata_ = metadata; return m }

// WithFlow can be used to add flow to a Msg
func (m *DBMsg) WithFlow(flow *courier.FlowReference) courier.Msg { m.Flow_ = flow; return m }

// WithAttachment can be used to append to the media urls for a message
func (m *DBMsg) WithAttachment(url string) courier.Msg {
	m.Attachments_ = append(m.Attachments_, url)
	return m
}

// WithURNAuth can be used to add a URN auth setting to a message
func (m *DBMsg) WithURNAuth(auth string) courier.Msg {
	m.URNAuth_ = auth
	return m
}

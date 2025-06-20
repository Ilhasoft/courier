1.6.0-courier-10.0.0
----------
  * Add new Messangi handler for message processing

1.5.13-courier-10.0.0
----------
  * Implement Markdown text escaping in Telegram handler and add corresponding tests

1.5.12-courier-10.0.0
----------
  * Update sentry-go and slog-sentry packages

1.5.11-courier-10.0.0
----------
  * Add Dynamo AWS region config variable

1.5.10-courier-10.0.0
----------
  * Update to v10.0.0

1.5.10-courier-9.2.0
----------
  * Update to v9.2.0

1.5.9-courier-9.0.1
----------
  * Update to v9.0.1
  * Update dockerfile

1.5.8-courier-8.2.1
----------
  * Remove menu button name mapping

1.5.7-courier-8.2.1
----------
  * Update to v8.2.1
  * Fix tests

1.5.6-courier-7.5.66
----------
  * Add support for sending templates with media in WA

1.5.5-courier-7.5.66
----------
  * Use failure status to avoid message retry

1.5.4-courier-7.5.66
----------
  * Remove errored status for kannel channel

1.5.3-courier-7.5.66
----------
  * Update to v7.5.66

1.5.3-courier-7.5.64
----------
  * Temporarily remove fault status for kannel channels

1.5.2-courier-7.5.64
----------
  * Update to v7.5.64

1.5.2-courier-7.4.0
----------
  * Add module to send webhooks for WAC and WA

1.5.1-courier-7.4.0
----------
  * Fix WAC handler
  * Fix: Remove last seen on 

1.5.0-courier-7.2.0
----------
  * Merge tag v7.2.0 from nyaruka into our 1.4.5-courier-7.1.0

1.4.8-courier-7.1.0
----------
  * Fix word 'menu' in Arabic for list messages #141

1.4.7-courier-7.1.0
----------
  * Add "Menu" word translation mapping to list messages in WAC and WA channels #139

1.4.6-courier-7.1.0
----------
  * Normalize quick response strings with slashes for TG and WA channels #137
  * Fix receiving multiple media for TG, WAC and WA channels #136

1.4.5-courier-7.1.0
----------
  * Remove expiration_timestamp from moPayload in WAC #133

1.4.4-courier-7.1.0
----------
  * Add support for sending captioned attachments in WAC #131
 
1.4.3-courier-7.1.0
----------
  * Quick Replies support in the Slack handler #129

1.4.2-courier-7.1.0
----------
  * Fix URL of attachments in WAC handler #127


1.4.1-courier-7.1.0
----------  
  * Fix receiving attachments and quick replies

1.4.0-courier-7.1.0
----------  
  * Integration support with Microsoft Teams

1.3.3-courier-7.1.0
----------  
  * Media message template support, link preview and document name correction on WhatsApp Cloud #118

1.3.2-courier-7.1.0
----------
  * Fix to prevent create a new contact without extra 9 in wpp number, instead, updating if already has one with the extra 9, handled in whatsapp cloud channels #119

1.3.1-courier-7.1.0
----------
  * Fix to ensure update last_seen_on if there is no error and no failure to send the message.

1.3.0-courier-7.1.0
----------
  * Slack Bot Channel Handler
  * Whatsapp Cloud Handler

1.2.1-courier-7.1.0
----------
  * Update contact last_seen_on on send message to him

1.2.0-courier-7.1.0
----------
  * Merge tag v7.1.0 from nyaruka into our 1.1.8-courier-7.0.0

1.1.8-courier-7.0.0
----------
 * Fix whatsapp handler to update the contact URN if the wa_id returned in the send message request is different from the current URN path, avoiding creating a new contact.

1.1.7-courier-7.0.0
----------
 * Add library with greater support for detection of mime types in Whatsapp

1.1.6-courier-7.0.0
----------
 * Support for viewing sent links in Whatsapp messages

1.1.5-courier-7.0.0
----------
 * Fix sending document names in whatsapp media message templates

1.1.4-courier-7.0.0
----------
 * Add Kyrgyzstan language support in whatsapp templates

1.1.3-courier-7.0.0
----------
 * fix whatsapp uploaded attachment file name

1.1.2-courier-7.0.0
----------
 * Fix metadata fetching for new Facebook contacts

1.1.1-courier-7.0.0
----------
 * Add Instagram Handler
 * Update gocommon to v1.16.2

1.1.0-courier-7.0.0
----------
 * Fix: Gujarati whatsapp language code
 * add button layout support on viber channel

1.0.0-courier-7.0.0
----------
 * Update Dockerfile to go 1.17.5 
 * Support to facebook customer feedback template
 * Support whatsapp media message template
 * Fix to prevent requests from blocked contact generate channel log
 * Weni-Webchat handler
 * Support to build Docker image

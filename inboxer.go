// Package inboxer is a Go library for checking email using the google Gmail
// API.

// Copyright (c) 2017 J. Hartsfield

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package inboxer

// SCOPE:
// TODO:
// channels and go routines
// Watch inbox
// metalinter vet etc
//
// tests (test for both messages and "threads")
// put ExampleFunctions in test file
// DOCS
// README.md
// how-to: add client credentials (for readme/docs)
// Get Previews/snippet (put in docs)
// spell checg
//
// WORKS:
// Mark as read/unread/important/spam
// Get emails by query
// Get email metadata
// Get email main body
// Get labels
// Check for unread messages
// Convert date to human readable format
//
// DONE:
// LICENSE
// check/return errors/fmt

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	gmail "google.golang.org/api/gmail/v1"
)

// MarkAs allows you to mark an email with a specific label using the
// gmail.ModifyMessageRequest struct.
func MarkAs(srv *gmail.Service, msg *gmail.Message, req *gmail.ModifyMessageRequest) (*gmail.Message, error) {
	return srv.Users.Messages.Modify("me", msg.Id, req).Do()
}

// MarkAllAsRead removes the UNREAD label from all emails.
func MarkAllAsRead(srv *gmail.Service) error {
	req := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}

	msgs, err := Query(srv, "label:UNREAD")
	if err != nil {
		return err
	}
	for _, v := range msgs {
		_, err := MarkAs(srv, v, req)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetBody gets, decodes, and returns the body of the email. It returns an
// error if decoding goes wrong. mimeType is used to indicate whether you want
// the plain text or html encoding ("text/html", "text/plain").
func GetBody(msg *gmail.Message, mimeType string) (string, error) {
	for _, v := range msg.Payload.Parts {
		if v.MimeType == "multipart/alternative" {
			for _, l := range v.Parts {
				if l.MimeType == mimeType && l.Body.Size >= 1 {
					dec, err := decodeEmailBody(l.Body.Data)
					if err != nil {
						return "", err
					}
					return dec, nil
				}
			}
		}
		if v.MimeType == mimeType && v.Body.Size >= 1 {
			dec, err := decodeEmailBody(v.Body.Data)
			if err != nil {
				return "", err
			}
			return dec, nil
		}
	}
	return "", errors.New("Couldn't Read Body")
}

// PartialMetadata stores email metadata. Some fields may sound redundant, but
// in fact have different contexts. Some are slices of string because the ones
// that have multiple values are still being sorted from those that don't.
type PartialMetadata struct {
	// Sender is the entity that originally created and sent the message
	Sender string
	// From is the entity that sent the message to you (e.g. googlegroups). Most
	// of the time this information is only relevant to mailing lists.
	From string
	// Subject is the email subject
	Subject string
	// Mailing list contains the name of the mailing list that the email was
	// posted to, if any.
	MailingList string
	// CC is the "carbon copy" list of addresses
	CC []string
	// To is the recipient of the email.
	To []string
	// ThreadTopic contains the topic of the thread (e.g. google groups threads)
	ThreadTopic []string
	// DeliveredTo is who the email was sent to. This can contain multiple
	// addresses if the email was forwarded.
	DeliveredTo []string
}

// GetPartialMetadata gets some of the useful metadata from the headers.
func GetPartialMetadata(msg *gmail.Message) *PartialMetadata {
	info := &PartialMetadata{}
	for _, v := range msg.Payload.Headers {
		switch v.Name {
		case "Sender":
			info.Sender = v.Value
		case "From":
			info.From = v.Value
		case "Subject":
			info.Subject = v.Value
		case "Mailing-list":
			info.MailingList = v.Value
		case "CC":
			info.CC = append(info.CC, v.Value)
		case "To":
			info.To = append(info.To, v.Value)
		case "Thread-Topic":
			info.ThreadTopic = append(info.ThreadTopic, v.Value)
		case "Delivered-To":
			info.DeliveredTo = append(info.DeliveredTo, v.Value)
		}
	}
	return info
}

// decodeEmailBody is used to decode the email body by converting from
// URLEncoded base64 to a string.
func decodeEmailBody(data string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// ReceivedTime parses and converts a Unix time stamp into a human readable
// format ().
func ReceivedTime(datetime int64) (time.Time, error) {
	conv := strconv.FormatInt(datetime, 10)
	// Remove trailing zeros.
	conv = conv[:len(conv)-3]
	tc, err := strconv.ParseInt(conv, 10, 64)
	if err != nil {
		return time.Unix(tc, 0), err
	}
	return time.Unix(tc, 0), nil
}

// Query queries the inbox for a string following the search style of the gmail
// online mailbox.
// example:
// "in:sent after:2017/01/01 before:2017/01/30"
func Query(srv *gmail.Service, query string) ([]*gmail.Message, error) {
	inbox, err := srv.Users.Messages.List("me").Q(query).Do()
	if err != nil {
		return inbox.Messages, err
	}
	msgs, err := getByID(srv, inbox)
	if err != nil {
		return msgs, err
	}
	return msgs, nil
}

// getByID gets emails individually by ID. This is necessary because this is
// how the gmail API is set [0][1] up apparently (but why?).
// [0] https://developers.google.com/gmail/api/v1/reference/users/messages/get
// [1] https://stackoverflow.com/questions/36365172/message-payload-is-always-null-for-all-messages-how-do-i-get-this-data
func getByID(srv *gmail.Service, msgs *gmail.ListMessagesResponse) ([]*gmail.Message, error) {
	var msgSlice []*gmail.Message
	for _, v := range msgs.Messages {
		msg, err := srv.Users.Messages.Get("me", v.Id).Do()
		if err != nil {
			return msgSlice, err
		}
		msgSlice = append(msgSlice, msg)
	}
	return msgSlice, nil
}

// GetMessages gets and returns gmail messages
func GetMessages(srv *gmail.Service, howMany uint) ([]*gmail.Message, error) {
	var msgSlice []*gmail.Message

	// Get the messages
	inbox, err := srv.Users.Messages.List("me").MaxResults(int64(howMany)).Do()
	if err != nil {
		return msgSlice, err
	}

	msgs, err := getByID(srv, inbox)
	if err != nil {
		return msgs, err
	}
	return msgs, nil
}

// CheckForUnreadByLabel checks for unread mail matching the specified label.
// NOTE: When checking your inbox for unread messages, it's not uncommon for
// it to return thousands of unread messages that you don't know about. To see
// them in gmail, query your mail for "label:unread". For CheckForUnreadByLabel
// to work properly you need to mark all mail as read either through gmail or
// through the MarkAllAsRead() function found in this library.
func CheckForUnreadByLabel(srv *gmail.Service, label string) (int64, error) {
	inbox, err := srv.Users.Labels.Get("me", label).Do()
	if err != nil {
		return -1, err
	}

	if inbox.MessagesUnread == 0 && inbox.ThreadsUnread == 0 {
		return 0, nil
	}

	return inbox.MessagesUnread + inbox.ThreadsUnread, nil
}

// CheckForUnread checks for mail labeled "UNREAD".
// NOTE: When checking your inbox for unread messages, it's not uncommon for
// it to return thousands of unread messages that you don't know about. To see
// them in gmail, query your mail for "label:unread". For CheckForUnread to
// work properly you need to mark all mail as read either through gmail or
// through the MarkAllAsRead() function found in this library.
func CheckForUnread(srv *gmail.Service) (int64, error) {
	inbox, err := srv.Users.Labels.Get("me", "UNREAD").Do()
	if err != nil {
		return -1, err
	}

	if inbox.MessagesUnread == 0 && inbox.ThreadsUnread == 0 {
		return 0, nil
	}

	return inbox.MessagesUnread + inbox.ThreadsUnread, nil
}

// GetLabels gets a list of the labels used in the users inbox.
func GetLabels(srv *gmail.Service) (*gmail.ListLabelsResponse, error) {
	return srv.Users.Labels.List("me").Do()
}

// WatchInbox watches the user inbox
func WatchInbox(srv *gmail.Service) {
	req := &gmail.WatchRequest{
		LabelFilterAction: "include",
		LabelIds:          []string{"UNREAD"},
		// projects/my-project-id/topics/my-topic-id
		TopicName: "projects/keen-vision-135323/topics/gmailmsg",
	}

	wr, err := srv.Users.Watch("me", req).Do()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(wr.ServerResponse.HTTPStatusCode)
}

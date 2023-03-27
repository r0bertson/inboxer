// Package inboxer is a Go library for checking email using the Google Gmail API.

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
// FITNESS FOR A PARTICULAR PURPOSE AND NON INFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package inboxer

import (
	"google.golang.org/api/gmail/v1"
)

type Service struct {
	GmailSvc *gmail.Service
}

func NewGmailService(credentialsFilePath string, scopes ...string) (*Service, error) {
	srv, err := GetGmailServiceFromFile(credentialsFilePath, scopes...)

	if err != nil {
		return nil, err
	}
	return &Service{srv}, nil
}

// MarkAs allows you to mark an email with a specific label using the gmail.ModifyMessageRequest struct.
func (s *Service) MarkAs(msg *gmail.Message, req *gmail.ModifyMessageRequest) (*gmail.Message, error) {
	return s.GmailSvc.Users.Messages.Modify("me", msg.Id, req).Do()
}

// MarkAllAsRead removes the UNREAD label from all emails.
func (s *Service) MarkAllAsRead() error {
	// Request to remove the label ID "UNREAD"
	req := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}

	// Get the messages labeled "UNREAD"
	msgs, err := s.Query("label:UNREAD")
	if err != nil {
		return err
	}

	// For each UNREAD message, request to remove the "UNREAD" label (thus marking it as "READ").
	for _, msg := range msgs {
		if _, err := s.MarkAs(msg, req); err != nil {
			return err
		}
	}

	return nil
}

// Query queries the inbox for a string following the search style of the gmail online mailbox.
// example: "in:sent after:2017/01/01 before:2017/01/30"
func (s *Service) Query(query string) ([]*gmail.Message, error) {
	inbox, err := s.GmailSvc.Users.Messages.List("me").Q(query).Do()
	if err != nil {
		return []*gmail.Message{}, err
	}
	msgs, err := s.MessageByID(inbox)
	if err != nil {
		return msgs, err
	}
	return msgs, nil
}

// MessageByID gets an email individually by ID. This is necessary because this is how the gmail API is set [0][1] up apparently (but why?).
// [0] https://developers.google.com/gmail/api/v1/reference/users/messages/get
// [1] https://stackoverflow.com/questions/36365172/message-payload-is-always-null-for-all-messages-how-do-i-get-this-data
func (s *Service) MessageByID(msgs *gmail.ListMessagesResponse) ([]*gmail.Message, error) {
	var msgSlice []*gmail.Message
	for _, v := range msgs.Messages {
		msg, err := s.GmailSvc.Users.Messages.Get("me", v.Id).Do()
		if err != nil {
			return msgSlice, err
		}
		msgSlice = append(msgSlice, msg)
	}
	return msgSlice, nil
}

// GetMessages gets and returns gmail messages
func (s *Service) GetMessages(howMany uint) ([]*gmail.Message, error) {
	var msgSlice []*gmail.Message

	// Get the messages
	inbox, err := s.GmailSvc.Users.Messages.List("me").MaxResults(int64(howMany)).Do()
	if err != nil {
		return msgSlice, err
	}

	msgs, err := s.MessageByID(inbox)
	if err != nil {
		return msgs, err
	}
	return msgs, nil
}

// CheckForUnread checks for mail labeled "UNREAD".
// NOTE: When checking your inbox for unread messages, it's not uncommon for
// it to return thousands of unread messages that you don't know about. To see
// them in gmail, query your mail for "label:unread". For CheckForUnread to
// work properly you need to mark all mail as read either through gmail or
// through the MarkAllAsRead() function found in this library.
func (s *Service) CheckForUnread() (int64, error) {
	inbox, err := s.GmailSvc.Users.Labels.Get("me", "UNREAD").Do()
	if err != nil {
		return -1, err
	}

	if inbox.MessagesUnread == 0 && inbox.ThreadsUnread == 0 {
		return 0, nil
	}

	return inbox.MessagesUnread + inbox.ThreadsUnread, nil
}

// GetLabels gets a list of the labels used in the users inbox.
func (s *Service) GetLabels() (*gmail.ListLabelsResponse, error) {
	return s.GmailSvc.Users.Labels.List("me").Do()
}

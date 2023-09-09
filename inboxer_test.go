package inboxer

import (
	qt "github.com/frankban/quicktest"
	"github.com/zeebo/assert"
	"google.golang.org/api/gmail/v1"
	"testing"
)

// run cmd/main.go first, because we need to go to browser
func TestInboxer(t *testing.T) {
	checker := qt.New(t)
	service, err := NewGmailService("./limatech-desktop-credentials.json", gmail.MailGoogleComScope)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	checker.Run("GetLabels", func(c *qt.C) {
		labels, err := service.GetLabels()
		c.Assert(err, qt.IsNil)
		c.Assert(labels, qt.IsNotNil)
	})

}

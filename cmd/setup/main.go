package main

import (
	"github.com/r0bertson/inboxer"
	"google.golang.org/api/gmail/v1"
	"log"
	"os"
)

// go run main.go credentials_filepath.json
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("you must pass your service account credentials as argument like: `go run main.go /Users/your_user/service_account_file.json`")
	}
	inboxer.SetupGmailService(args[0], gmail.MailGoogleComScope)
}

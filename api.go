package inboxer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"google.golang.org/api/gmail/v1"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const TokenFile = "gmail-token.json"

// SetupGmailService sets a token file if not already present. This needs human intervention, so it is advised
// to run the application at /cmd/setup directory before using this lib.
func SetupGmailService(credentialsPath string, scope ...string) error {
	cacheFile, err := newTokenizer()
	if err != nil {
		return err
	}
	if _, err = tokenFromFile(cacheFile); err == nil {
		log.Println("gmail service credentials already set")
		return nil
	}

	credentialsFile, err := os.ReadFile(credentialsPath)
	if err != nil {
		return err
	}

	config, err := google.ConfigFromJSON(credentialsFile, scope...)
	if err != nil {
		return err
	}

	saveToken(cacheFile, getTokenFromWeb(config))
	log.Println("gmail service credentials set")
	return nil
}

// GetGmailServiceFromFile will use a credentials file and a token file set to build a gmail.Service instance
// if one of the files are not present, this function will return an error.
func GetGmailServiceFromFile(credentialsPath string, scope ...string) (*gmail.Service, error) {
	credentialsFile, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(credentialsFile, scope...)
	if err != nil {
		return nil, err
	}

	cacheFile, err := newTokenizer()
	if err != nil {
		return nil, err
	}
	token, err := tokenFromFile(cacheFile)
	if err != nil {
		return nil, err
	}
	return gmail.New(config.Client(context.Background(), token))
}

// newTokenizer returns a new token and generates credential file path and
// returns the generated credential path/filename along with any errors.
func newTokenizer() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir, url.QueryEscape(TokenFile)), nil
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read errors encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

// getTokenFromWeb uses Config to request a Token. It returns the retrieved
// Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var code string
	fmt.Print("Type the code you got on the URL: ")
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("unable to retrieve token from web %v", err)
	}
	return tok
}

// saveToken uses a file path to create a file and store the token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

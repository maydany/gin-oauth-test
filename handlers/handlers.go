package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/nodebreaker0-0/gin-oauth-test/structs"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var cred Credentials
var conf *oauth2.Config

// Credentials which stores google ids.
type Credentials struct {
	Cid         string `json:"cid"`
	Csecret     string `json:"csecret"`
	RedirectURL string `json:"url"`
}

// RandToken generates a random @l length token.
func RandToken(l int) (string, error) {
	b := make([]byte, l)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func getLoginURL(state string) string {
	return conf.AuthCodeURL(state)
}

func init() {
	file, err := ioutil.ReadFile("./creds.json")
	if err != nil {
		log.Printf("File error: %v\n", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(file, &cred); err != nil {
		log.Println("unable to marshal data")
		return
	}

	conf = &oauth2.Config{
		ClientID:     cred.Cid,
		ClientSecret: cred.Csecret,
		RedirectURL:  cred.RedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email", // You have to select your own scope from here -> https://developers.google.com/identity/protocols/googlescopes#google_sign-in
		},
		Endpoint: google.Endpoint,
	}
}

// AuthHandler handles authentication of a user and initiates a session.
func AuthHandler(c *gin.Context) {
	// Handle the exchange code to initiate a transport.
	session := sessions.Default(c)
	retrievedState := session.Get("state")
	queryState := c.Request.URL.Query().Get("state")
	if retrievedState != queryState {
		log.Printf("Invalid session state: retrieved: %s; Param: %s", retrievedState, queryState)
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{"message": "Invalid session state."})
		return
	}
	code := c.Request.URL.Query().Get("code")
	tok, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{"message": "Login failed. Please try again."})
		return
	}

	client := conf.Client(oauth2.NoContext, tok)
	userinfo, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	defer userinfo.Body.Close()
	data, _ := ioutil.ReadAll(userinfo.Body)
	u := structs.User{}
	if err = json.Unmarshal(data, &u); err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{"message": "Error marshalling response. Please try agian."})
		return
	}
	session.Set("user-id", u.Email)
	err = session.Save()
	if err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{"message": "Error while saving session. Please try again."})
		return
	}
	c.HTML(http.StatusOK, "validator.tmpl", gin.H{"link": "/"})
}

// LoginHandler handles the login procedure.
func LoginHandler(c *gin.Context) {
	state, err := RandToken(32)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{"message": "Error while generating random data."})
		return
	}
	session := sessions.Default(c)
	session.Set("state", state)
	err = session.Save()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{"message": "Error while saving session."})
		return
	}
	link := getLoginURL(state)
	c.HTML(http.StatusOK, "auth.tmpl", gin.H{"link": link})
}

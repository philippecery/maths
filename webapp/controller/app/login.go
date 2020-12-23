package app

import (
	"bytes"
	hash "crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"github.com/philippecery/maths/webapp/constant"
	"github.com/philippecery/maths/webapp/database/dataaccess"
	"github.com/philippecery/maths/webapp/database/document"
	"github.com/philippecery/maths/webapp/session"
)

// Login handles requests to /login
// Only GET and POST requests are allowed.
//  - a GET request will display the Login form. If an error message is available in the session, it will be displayed.
//  - a POST request will authenticate the user if the submitted credentials are valid.
func Login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/login")
	httpsession := session.GetSession(w, r)
	if r.Method == "GET" {
		vd := newViewData(w, r)
		vd.setErrorMessage(httpsession.GetErrorMessageID())
		vd.setToken(httpsession.SetCSRFToken())
		vd.setLoginPageLocalizedMessages()
		if err := templates.ExecuteTemplate(w, "login.html.tpl", vd); err != nil {
			log.Fatalf("Error while executing template 'login': %v\n", err)
		}
	} else {
		if r.Method == "POST" {
			userID := r.PostFormValue("userId")
			if r.PostFormValue("token") == httpsession.GetCSRFToken() {
				if user := verifyUserIDPassword(userID, r.PostFormValue("password")); user != nil {
					session.InvalidateSession(w, r)
					httpsession := session.GetSession(w, r)
					httpsession.SetAuthenticatedUser(user)
					httpsession.SetCSRFToken()
					dataaccess.UpdateLastConnection(userID)
					if user.IsAdmin() {
						http.Redirect(w, r, "/admin/users", http.StatusFound)
					} else {
						http.Redirect(w, r, "/", http.StatusFound)
					}
					return
				}
				log.Printf("User %s: wrong password\n", userID)
			} else {
				log.Printf("User %s: wrong CSRF token\n", userID)
			}
		} else {
			log.Printf("Invalid method %s\n", r.Method)
		}
		httpsession.SetErrorMessageID("errorAuthenticationFailed")
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func (vd viewData) setLoginPageLocalizedMessages() viewData {
	return vd.setDefaultLocalizedMessages().
		addLocalizedMessage("login").
		addLocalizedMessage("userid").
		addLocalizedMessage("password")
}

func verifyUserIDPassword(userID, password string) *document.User {
	if user := dataaccess.GetUserByID(userID); user != nil {
		if hashedPwd, err := base64.StdEncoding.DecodeString(user.Password); err == nil {
			h := hash.New()
			h.Write(hashedPwd[:32])
			h.Write([]byte(password))
			if bytes.Equal(h.Sum(nil), hashedPwd[32:]) && user.Status == constant.Enabled {
				// TODO: reset number of attempts to 0
				return user
			}
			// TODO: increment number of attempts and update status if more than 5 failed attempts.
		}
	}
	return nil
}

// Logout handles requests to /logout by invalidating the session and redirecting to /login
func Logout(w http.ResponseWriter, r *http.Request) {
	session.InvalidateSession(w, r)
	http.Redirect(w, r, "/login", http.StatusFound)
}

package app

import (
	"bytes"
	"crypto/hmac"
	hash "crypto/sha256"
	"encoding/base64"
	"errors"
	"log"
	"net/http"

	"github.com/philippecery/maths/webapp/config"
	"github.com/philippecery/maths/webapp/database/dataaccess"
	"github.com/philippecery/maths/webapp/session"
)

// Users handles requests to /admin/users
// Only GET requests are allowed. The user must have role Admin to access this page.
// Displays the Users page with the list of registerd and unregistered users.
func Users(w http.ResponseWriter, r *http.Request) {
	httpsession := session.GetSession(w, r)
	if user := httpsession.GetAuthenticatedUser(); user != nil && user.IsAdmin() {
		vd := newViewData(w, r)
		vd.setUser(user)
		vd.setRegisteredUsers(dataaccess.GetAllRegisteredUsers())
		vd.setUnregisteredUsers(dataaccess.GetAllUnregisteredUsers())
		vd.setUsersPageLocalizedMessages()
		if err := templates.ExecuteTemplate(w, "users.html.tpl", vd); err != nil {
			log.Fatalf("Error while executing template 'users': %v\n", err)
		}
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (vd viewData) setUsersPageLocalizedMessages() viewData {
	return vd.setDefaultLocalizedMessages().
		addLocalizedMessage("registeredUsers").
		addLocalizedMessage("unregisteredUsers").
		addLocalizedMessage("userid").
		addLocalizedMessage("firstName").
		addLocalizedMessage("lastName").
		addLocalizedMessage("emailAddress").
		addLocalizedMessage("role").
		addLocalizedMessage("lastConnection").
		addLocalizedMessage("disableAccount").
		addLocalizedMessage("enableAccount").
		addLocalizedMessage("deleteUser").
		addLocalizedMessage("token").
		addLocalizedMessage("expires").
		addLocalizedMessage("copyRegistrationLink").
		addLocalizedMessage("addUser").
		addLocalizedMessage("logout")
}

// Status handles requests to /admin/status
// Only GET requests are allowed. The user must have role Admin to access this page.
// Toggles the status of the selected user if the token is valid
func Status(w http.ResponseWriter, r *http.Request) {
	executeAction(w, r, func() error {
		if err := dataaccess.ToggleUserStatus(r.URL.Query()["userid"][0]); err != nil {
			return errors.New("errorUserStatusUpdateFailed")
		}
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return nil
	})
}

// Delete handles requests to /admin/delete
// Only GET requests are allowed. The user must have role Admin to access this page.
// Deletes the selected user if the token is valid
func Delete(w http.ResponseWriter, r *http.Request) {
	executeAction(w, r, func() error {
		if err := dataaccess.DeleteUser(r.URL.Query()["userid"][0]); err != nil {
			return errors.New("errorUserDeletionFailed")
		}
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return nil
	})
}

func executeAction(w http.ResponseWriter, r *http.Request, action func() error) {
	httpsession := session.GetSession(w, r)
	if user := httpsession.GetAuthenticatedUser(); user != nil && user.IsAdmin() {
		if r.Method == "GET" {
			userID := r.URL.Query()["userid"][0]
			actionToken := r.URL.Query()["rnd"][0]
			if userID != "" && actionToken != "" && verifyActionToken(userID, actionToken) {
				var err error
				if err = action(); err != nil {
					httpsession.SetErrorMessageID(err.Error())
				}
				return
			}
			log.Println("Invalid userID or token")
		} else {
			log.Printf("Invalid method %s\n", r.Method)
		}
	} else {
		log.Println("User is not authenticated or does not have Admin role")
	}
	log.Println("Redirecting to Login page")
	http.Redirect(w, r, "/login", http.StatusFound)
}

func verifyActionToken(userID, actionToken string) bool {
	if token, err := base64.URLEncoding.DecodeString(actionToken); err == nil {
		mac := hmac.New(hash.New, []byte(config.Config.Keys.ActionToken))
		mac.Write([]byte(userID))
		mac.Write(token[:32])
		return bytes.Equal(token[32:], mac.Sum(nil))
	}
	return false
}

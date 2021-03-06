package public

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/philippecery/maths/webapp/constant/team"

	"github.com/philippecery/maths/webapp/config"
	"github.com/philippecery/maths/webapp/constant/user"
	"github.com/philippecery/maths/webapp/database/dataaccess"
	"github.com/philippecery/maths/webapp/database/model"
	"github.com/philippecery/maths/webapp/i18n"
	"github.com/philippecery/maths/webapp/services"
	"github.com/philippecery/maths/webapp/services/email"
	"github.com/philippecery/maths/webapp/session"
	"github.com/philippecery/maths/webapp/util"
)

// SignUp handles requests to /signup
// Only GET and POST requests are allowed. The user must be anonymous to access this page.
//  - a GET request will display the SignUp form. If an error message is available in the session, it will be displayed.
//  - a POST request will create a temporary user account if the submitted data are valid. That new account will have a token. The registration email is sent to the user.
func SignUp(w http.ResponseWriter, r *http.Request, httpsession *session.HTTPSession) {
	if r.Method == "GET" {
		vd := services.NewViewData(w, r)
		vd.SetErrorMessage(httpsession.GetErrorMessageID())
		vd.SetToken(httpsession.GetCSRFToken())
		vd.SetDefaultLocalizedMessages().
			AddLocalizedMessage("signup").
			AddLocalizedMessage("emailAddress").
			AddLocalizedMessage("save").
			AddLocalizedMessage("cancel")
		if err := services.Templates.ExecuteTemplate(w, "signup.html.tmpl", vd); err != nil {
			log.Fatalf("Error while executing template 'signup': %v\n", err)
		}
		return
	}
	if r.Method == "POST" {
		if r.PostFormValue("token") == httpsession.GetCSRFToken() {
			var err error
			var newTeam *model.Team
			var emailAddress string
			if newTeam, emailAddress, err = validateSignupUserInput(r); err == nil {

			}
			var userID string
			var protectedEmailAddress *model.PII
			if emailAddress, err = services.ValidateEmailAddress(strings.ToLower(r.PostFormValue("emailAddress"))); err == nil {
				if userID, err = util.ProtectUserID(emailAddress); err == nil {
					if existingUser := dataaccess.GetUserByID(userID); existingUser != nil {
						if err = email.SendAlreadyRegisteredEmail(services.NewEmailViewData(w, r), existingUser); err != nil {
							log.Printf("Unable to send email to existing user. Cause: %v", err)
						}
					} else {
						token, expirationTime := util.GenerateUserToken(userID)
						if protectedEmailAddress, err = model.Protect(emailAddress); err == nil {
							newUser := &model.User{UserID: userID, EmailAddress: protectedEmailAddress, Token: token, Expires: expirationTime, Role: user.ParentOrTeacher, Status: user.Unregistered}
							if err = dataaccess.CreateNewTeam(newTeam, newUser); err == nil {
								if err = sendSignUpEmail(services.NewEmailViewData(w, r), newUser); err != nil {
									log.Printf("Unable to send registration email to new user. Cause: %v", err)
								}
							} else {
								log.Printf("User creation failed. Cause: %v", err)
								httpsession.SetErrorMessageID("errorUserCreationFailed")
							}
						} else {
							log.Printf("/signup: Error while protecting email address: %v", err)
							httpsession.SetErrorMessageID("errorGenericMessage")
						}
					}
					if err == nil {
						if err = services.Templates.ExecuteTemplate(w, "signupConfirmation.html.tmpl", nil); err != nil {
							log.Printf("Error while executing template 'signupConfirmation': %v\n", err)
						}
						return
					}
				} else {
					log.Printf("/signup: Error while protecting userid: %v", err)
					httpsession.SetErrorMessageID("errorGenericMessage")
				}
			} else {
				log.Println("/signup: Invalid email address")
				httpsession.SetErrorMessageID(err.Error())
			}
		} else {
			log.Println("/signup: Invalid CSRF token")
			httpsession.SetErrorMessageID("errorGenericMessage")
		}
	} else {
		log.Printf("/signup: Invalid method %s\n", r.Method)
		httpsession.SetErrorMessageID("errorGenericMessage")
	}
	http.Redirect(w, r, "/signup", http.StatusFound)
}

func sendSignUpEmail(vd services.ViewData, unregisteredUser *model.User) error {
	vd.SetViewData("URL", unregisteredUser.Link())
	vd.SetEmailDefaultLocalizedMessages().
		AddLocalizedMessage("emailSignUpTitle").
		AddLocalizedMessage("emailSignUpPreHeader").
		AddLocalizedMessage("emailSignUpMessage1").
		AddLocalizedMessage("emailSignUpMessage2").
		AddLocalizedMessage("emailSignUpContinueRegistration").
		AddLocalizedMessage("emailSignUpLinkWillExpire", config.Config.UserTokenValidity, map[string]interface{}{
			"nbHours": config.Config.UserTokenValidity,
		})
	return email.Send(unregisteredUser.EmailAddress.Reveal(), "", i18n.GetLocalizedMessage(vd.GetCurrentLanguage(), "emailSignUpSubject"), "signup.email.html.tmpl", vd)
}

func validateSignupUserInput(r *http.Request) (*model.Team, string, error) {
	var err error
	var emailAddress string
	newTeam := &model.Team{}
	if typeID, _ := strconv.Atoi(r.PostFormValue("type")); typeID > 0 && team.Type(typeID).IsValid() {
		newTeam.Type = team.Type(typeID)
		newTeam.Name = r.PostFormValue("name")
		if len(newTeam.Name) >= 3 && len(newTeam.Name) <= 50 {
			if emailAddress, err = services.ValidateEmailAddress(strings.ToLower(r.PostFormValue("emailAddress"))); err == nil {
				newTeam.Language = i18n.ValidateLanguage(r.PostFormValue("lang"))
				newTeam.TeamID = util.GenerateUUID()
				return newTeam, emailAddress, nil
			}
		}
	}
	return nil, "", err
}

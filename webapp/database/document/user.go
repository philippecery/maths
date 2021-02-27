package document

import (
	"encoding/base64"
	"time"

	"github.com/philippecery/libs/bytes"
	"github.com/philippecery/libs/cipher"
	"github.com/philippecery/maths/webapp/config"
	"github.com/philippecery/maths/webapp/constant/user"
)

type PII string

// User document
type User struct {
	UserID          string
	Password        string
	PasswordDate    time.Time
	Name            string
	Language        string
	EmailAddress    PII
	EmailAddressTmp PII
	Role            user.Role
	Status          user.Status
	LastConnection  time.Time
	Token           string
	Expires         time.Time
	GradeID         string
	FailedAttempts  int
}

// UserProfile document
type UserProfile struct {
	UserID         string
	EmailAddress   PII
	Name           string
	Language       string
	LastConnection time.Time
}

// RegisteredUser document
// Replaces existing User document when user registers
type RegisteredUser struct {
	UserID         string
	EmailAddress   PII
	Password       string
	Name           string
	Language       string
	Role           user.Role
	Status         user.Status
	LastConnection time.Time
	PasswordDate   time.Time
}

// Student document
// Updates an existing User document
type Student struct {
	UserID   string
	Name     string
	Language string
	GradeID  string
	Grade    *Grade
}

// Link returns the registration link
func (u *User) Link() string {
	return "https://" + config.Config.Hostname + "/register?token=" + u.Token
}

// Enabled returns true if this user's status is Enabled.
func (u *User) Enabled() bool {
	return u.Status == user.Enabled
}

// IsAdmin return true is this user's role is Admin
func (u *User) IsAdmin() bool {
	return u.Role == user.Admin
}

// IsTeacher return true is this user's role is Teacher
func (u *User) IsTeacher() bool {
	return u.Role == user.Teacher
}

// IsStudent return true is this user's role is Student
func (u *User) IsStudent() bool {
	return u.Role == user.Student
}

// IsUnregistered return true is this user's status is Unregistered
func (u *User) IsUnregistered() bool {
	return u.Status == user.Unregistered
}

// IsDisabled return true is this user's status is Disabled
func (u *User) IsDisabled() bool {
	return u.Status == user.Disabled
}

func (p PII) Reveal() string {
	var err error
	var piiKey, protectedPIIBytes, piiBytes []byte
	piiKey, err = base64.StdEncoding.DecodeString(config.Config.Keys.PII)
	if err == nil {
		protectedPIIBytes, err = base64.StdEncoding.DecodeString(string(p))
		if err == nil {
			piiBytes, err = cipher.Decrypt(&piiKey, protectedPIIBytes)
			defer bytes.Clear(&piiBytes)
			return string(piiBytes)
		}
	}
	return ""
}

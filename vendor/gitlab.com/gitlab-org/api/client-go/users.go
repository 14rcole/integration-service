//
// Copyright 2021, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type (
	UsersServiceInterface interface {
		ListUsers(opt *ListUsersOptions, options ...RequestOptionFunc) ([]*User, *Response, error)
		GetUser(user int, opt GetUsersOptions, options ...RequestOptionFunc) (*User, *Response, error)
		CreateUser(opt *CreateUserOptions, options ...RequestOptionFunc) (*User, *Response, error)
		ModifyUser(user int, opt *ModifyUserOptions, options ...RequestOptionFunc) (*User, *Response, error)
		DeleteUser(user int, options ...RequestOptionFunc) (*Response, error)
		CurrentUser(options ...RequestOptionFunc) (*User, *Response, error)
		CurrentUserStatus(options ...RequestOptionFunc) (*UserStatus, *Response, error)
		GetUserStatus(uid any, options ...RequestOptionFunc) (*UserStatus, *Response, error)
		SetUserStatus(opt *UserStatusOptions, options ...RequestOptionFunc) (*UserStatus, *Response, error)
		GetUserAssociationsCount(user int, options ...RequestOptionFunc) (*UserAssociationsCount, *Response, error)
		ListSSHKeys(opt *ListSSHKeysOptions, options ...RequestOptionFunc) ([]*SSHKey, *Response, error)
		ListSSHKeysForUser(uid any, opt *ListSSHKeysForUserOptions, options ...RequestOptionFunc) ([]*SSHKey, *Response, error)
		GetSSHKey(key int, options ...RequestOptionFunc) (*SSHKey, *Response, error)
		GetSSHKeyForUser(user int, key int, options ...RequestOptionFunc) (*SSHKey, *Response, error)
		AddSSHKey(opt *AddSSHKeyOptions, options ...RequestOptionFunc) (*SSHKey, *Response, error)
		AddSSHKeyForUser(user int, opt *AddSSHKeyOptions, options ...RequestOptionFunc) (*SSHKey, *Response, error)
		DeleteSSHKey(key int, options ...RequestOptionFunc) (*Response, error)
		DeleteSSHKeyForUser(user, key int, options ...RequestOptionFunc) (*Response, error)
		ListGPGKeys(options ...RequestOptionFunc) ([]*GPGKey, *Response, error)
		GetGPGKey(key int, options ...RequestOptionFunc) (*GPGKey, *Response, error)
		AddGPGKey(opt *AddGPGKeyOptions, options ...RequestOptionFunc) (*GPGKey, *Response, error)
		DeleteGPGKey(key int, options ...RequestOptionFunc) (*Response, error)
		ListGPGKeysForUser(user int, options ...RequestOptionFunc) ([]*GPGKey, *Response, error)
		GetGPGKeyForUser(user, key int, options ...RequestOptionFunc) (*GPGKey, *Response, error)
		AddGPGKeyForUser(user int, opt *AddGPGKeyOptions, options ...RequestOptionFunc) (*GPGKey, *Response, error)
		DeleteGPGKeyForUser(user, key int, options ...RequestOptionFunc) (*Response, error)
		ListEmails(options ...RequestOptionFunc) ([]*Email, *Response, error)
		ListEmailsForUser(user int, opt *ListEmailsForUserOptions, options ...RequestOptionFunc) ([]*Email, *Response, error)
		GetEmail(email int, options ...RequestOptionFunc) (*Email, *Response, error)
		AddEmail(opt *AddEmailOptions, options ...RequestOptionFunc) (*Email, *Response, error)
		AddEmailForUser(user int, opt *AddEmailOptions, options ...RequestOptionFunc) (*Email, *Response, error)
		DeleteEmail(email int, options ...RequestOptionFunc) (*Response, error)
		DeleteEmailForUser(user, email int, options ...RequestOptionFunc) (*Response, error)
		BlockUser(user int, options ...RequestOptionFunc) error
		UnblockUser(user int, options ...RequestOptionFunc) error
		BanUser(user int, options ...RequestOptionFunc) error
		UnbanUser(user int, options ...RequestOptionFunc) error
		DeactivateUser(user int, options ...RequestOptionFunc) error
		ActivateUser(user int, options ...RequestOptionFunc) error
		ApproveUser(user int, options ...RequestOptionFunc) error
		RejectUser(user int, options ...RequestOptionFunc) error
		GetAllImpersonationTokens(user int, opt *GetAllImpersonationTokensOptions, options ...RequestOptionFunc) ([]*ImpersonationToken, *Response, error)
		GetImpersonationToken(user, token int, options ...RequestOptionFunc) (*ImpersonationToken, *Response, error)
		CreateImpersonationToken(user int, opt *CreateImpersonationTokenOptions, options ...RequestOptionFunc) (*ImpersonationToken, *Response, error)
		RevokeImpersonationToken(user, token int, options ...RequestOptionFunc) (*Response, error)
		CreatePersonalAccessToken(user int, opt *CreatePersonalAccessTokenOptions, options ...RequestOptionFunc) (*PersonalAccessToken, *Response, error)
		CreatePersonalAccessTokenForCurrentUser(opt *CreatePersonalAccessTokenForCurrentUserOptions, options ...RequestOptionFunc) (*PersonalAccessToken, *Response, error)
		GetUserActivities(opt *GetUserActivitiesOptions, options ...RequestOptionFunc) ([]*UserActivity, *Response, error)
		GetUserMemberships(user int, opt *GetUserMembershipOptions, options ...RequestOptionFunc) ([]*UserMembership, *Response, error)
		DisableTwoFactor(user int, options ...RequestOptionFunc) error
		CreateUserRunner(opts *CreateUserRunnerOptions, options ...RequestOptionFunc) (*UserRunner, *Response, error)
		CreateServiceAccountUser(opts *CreateServiceAccountUserOptions, options ...RequestOptionFunc) (*User, *Response, error)
		ListServiceAccounts(opt *ListServiceAccountsOptions, options ...RequestOptionFunc) ([]*ServiceAccount, *Response, error)
		UploadAvatar(avatar io.Reader, filename string, options ...RequestOptionFunc) (*User, *Response, error)
		DeleteUserIdentity(user int, provider string, options ...RequestOptionFunc) (*Response, error)

		// events.go
		ListUserContributionEvents(uid any, opt *ListContributionEventsOptions, options ...RequestOptionFunc) ([]*ContributionEvent, *Response, error)
	}

	// UsersService handles communication with the user related methods of
	// the GitLab API.
	//
	// GitLab API docs: https://docs.gitlab.com/api/users/
	UsersService struct {
		client *Client
	}
)

var _ UsersServiceInterface = (*UsersService)(nil)

// List a couple of standard errors.
var (
	ErrUserActivatePrevented         = errors.New("cannot activate a user that is blocked by admin or by LDAP synchronization")
	ErrUserApprovePrevented          = errors.New("cannot approve a user that is blocked by admin or by LDAP synchronization")
	ErrUserBlockPrevented            = errors.New("cannot block a user that is already blocked by LDAP synchronization")
	ErrUserConflict                  = errors.New("user does not have a pending request")
	ErrUserDeactivatePrevented       = errors.New("cannot deactivate a user that is blocked by admin or by LDAP synchronization")
	ErrUserDisableTwoFactorPrevented = errors.New("cannot disable two factor authentication if not authenticated as administrator")
	ErrUserNotFound                  = errors.New("user does not exist")
	ErrUserRejectPrevented           = errors.New("cannot reject a user if not authenticated as administrator")
	ErrUserTwoFactorNotEnabled       = errors.New("cannot disable two factor authentication if not enabled")
	ErrUserUnblockPrevented          = errors.New("cannot unblock a user that is blocked by LDAP synchronization")
)

// BasicUser included in other service responses (such as merge requests, pipelines, etc).
type BasicUser struct {
	ID        int        `json:"id"`
	Username  string     `json:"username"`
	Name      string     `json:"name"`
	State     string     `json:"state"`
	Locked    bool       `json:"locked"`
	CreatedAt *time.Time `json:"created_at"`
	AvatarURL string     `json:"avatar_url"`
	WebURL    string     `json:"web_url"`
}

// ServiceAccount represents a GitLab service account.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_service_accounts/
type ServiceAccount struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// User represents a GitLab user.
//
// GitLab API docs: https://docs.gitlab.com/api/users/
type User struct {
	ID                             int                `json:"id"`
	Username                       string             `json:"username"`
	Email                          string             `json:"email"`
	Name                           string             `json:"name"`
	State                          string             `json:"state"`
	WebURL                         string             `json:"web_url"`
	CreatedAt                      *time.Time         `json:"created_at"`
	Bio                            string             `json:"bio"`
	Bot                            bool               `json:"bot"`
	Location                       string             `json:"location"`
	PublicEmail                    string             `json:"public_email"`
	Skype                          string             `json:"skype"`
	Linkedin                       string             `json:"linkedin"`
	Twitter                        string             `json:"twitter"`
	WebsiteURL                     string             `json:"website_url"`
	Organization                   string             `json:"organization"`
	JobTitle                       string             `json:"job_title"`
	ExternUID                      string             `json:"extern_uid"`
	Provider                       string             `json:"provider"`
	ThemeID                        int                `json:"theme_id"`
	LastActivityOn                 *ISOTime           `json:"last_activity_on"`
	ColorSchemeID                  int                `json:"color_scheme_id"`
	IsAdmin                        bool               `json:"is_admin"`
	IsAuditor                      bool               `json:"is_auditor"`
	AvatarURL                      string             `json:"avatar_url"`
	CanCreateGroup                 bool               `json:"can_create_group"`
	CanCreateProject               bool               `json:"can_create_project"`
	ProjectsLimit                  int                `json:"projects_limit"`
	CurrentSignInAt                *time.Time         `json:"current_sign_in_at"`
	CurrentSignInIP                *net.IP            `json:"current_sign_in_ip"`
	LastSignInAt                   *time.Time         `json:"last_sign_in_at"`
	LastSignInIP                   *net.IP            `json:"last_sign_in_ip"`
	ConfirmedAt                    *time.Time         `json:"confirmed_at"`
	TwoFactorEnabled               bool               `json:"two_factor_enabled"`
	Note                           string             `json:"note"`
	Identities                     []*UserIdentity    `json:"identities"`
	External                       bool               `json:"external"`
	PrivateProfile                 bool               `json:"private_profile"`
	SharedRunnersMinutesLimit      int                `json:"shared_runners_minutes_limit"`
	ExtraSharedRunnersMinutesLimit int                `json:"extra_shared_runners_minutes_limit"`
	UsingLicenseSeat               bool               `json:"using_license_seat"`
	CustomAttributes               []*CustomAttribute `json:"custom_attributes"`
	NamespaceID                    int                `json:"namespace_id"`
	Locked                         bool               `json:"locked"`
	CreatedBy                      *BasicUser         `json:"created_by"`
}

// UserIdentity represents a user identity.
type UserIdentity struct {
	Provider  string `json:"provider"`
	ExternUID string `json:"extern_uid"`
}

// UserAvatar represents a GitLab user avatar.
//
// GitLab API docs: https://docs.gitlab.com/api/users/
type UserAvatar struct {
	Filename string
	Image    io.Reader
}

// MarshalJSON implements the json.Marshaler interface.
func (a *UserAvatar) MarshalJSON() ([]byte, error) {
	if a.Filename == "" && a.Image == nil {
		return []byte(`""`), nil
	}
	type alias UserAvatar
	return json.Marshal((*alias)(a))
}

// ListUsersOptions represents the available ListUsers() options.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#list-users
type ListUsersOptions struct {
	ListOptions
	Active          *bool `url:"active,omitempty" json:"active,omitempty"`
	Blocked         *bool `url:"blocked,omitempty" json:"blocked,omitempty"`
	Humans          *bool `url:"humans,omitempty" json:"humans,omitempty"`
	ExcludeInternal *bool `url:"exclude_internal,omitempty" json:"exclude_internal,omitempty"`
	ExcludeActive   *bool `url:"exclude_active,omitempty" json:"exclude_active,omitempty"`
	ExcludeExternal *bool `url:"exclude_external,omitempty" json:"exclude_external,omitempty"`
	ExcludeHumans   *bool `url:"exclude_humans,omitempty" json:"exclude_humans,omitempty"`

	// The options below are only available for admins.
	Search               *string    `url:"search,omitempty" json:"search,omitempty"`
	Username             *string    `url:"username,omitempty" json:"username,omitempty"`
	ExternalUID          *string    `url:"extern_uid,omitempty" json:"extern_uid,omitempty"`
	Provider             *string    `url:"provider,omitempty" json:"provider,omitempty"`
	CreatedBefore        *time.Time `url:"created_before,omitempty" json:"created_before,omitempty"`
	CreatedAfter         *time.Time `url:"created_after,omitempty" json:"created_after,omitempty"`
	OrderBy              *string    `url:"order_by,omitempty" json:"order_by,omitempty"`
	Sort                 *string    `url:"sort,omitempty" json:"sort,omitempty"`
	TwoFactor            *string    `url:"two_factor,omitempty" json:"two_factor,omitempty"`
	Admins               *bool      `url:"admins,omitempty" json:"admins,omitempty"`
	External             *bool      `url:"external,omitempty" json:"external,omitempty"`
	WithoutProjects      *bool      `url:"without_projects,omitempty" json:"without_projects,omitempty"`
	WithCustomAttributes *bool      `url:"with_custom_attributes,omitempty" json:"with_custom_attributes,omitempty"`
	WithoutProjectBots   *bool      `url:"without_project_bots,omitempty" json:"without_project_bots,omitempty"`
}

// ListUsers gets a list of users.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#list-users
func (s *UsersService) ListUsers(opt *ListUsersOptions, options ...RequestOptionFunc) ([]*User, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "users", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var usr []*User
	resp, err := s.client.Do(req, &usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, nil
}

// GetUsersOptions represents the available GetUser() options.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#get-a-single-user
type GetUsersOptions struct {
	WithCustomAttributes *bool `url:"with_custom_attributes,omitempty" json:"with_custom_attributes,omitempty"`
}

// GetUser gets a single user.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#get-a-single-user
func (s *UsersService) GetUser(user int, opt GetUsersOptions, options ...RequestOptionFunc) (*User, *Response, error) {
	u := fmt.Sprintf("users/%d", user)

	req, err := s.client.NewRequest(http.MethodGet, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, nil
}

// CreateUserOptions represents the available CreateUser() options.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#create-a-user
type CreateUserOptions struct {
	Admin               *bool       `url:"admin,omitempty" json:"admin,omitempty"`
	Avatar              *UserAvatar `url:"-" json:"-"`
	Bio                 *string     `url:"bio,omitempty" json:"bio,omitempty"`
	CanCreateGroup      *bool       `url:"can_create_group,omitempty" json:"can_create_group,omitempty"`
	Email               *string     `url:"email,omitempty" json:"email,omitempty"`
	External            *bool       `url:"external,omitempty" json:"external,omitempty"`
	ExternUID           *string     `url:"extern_uid,omitempty" json:"extern_uid,omitempty"`
	ForceRandomPassword *bool       `url:"force_random_password,omitempty" json:"force_random_password,omitempty"`
	JobTitle            *string     `url:"job_title,omitempty" json:"job_title,omitempty"`
	Linkedin            *string     `url:"linkedin,omitempty" json:"linkedin,omitempty"`
	Location            *string     `url:"location,omitempty" json:"location,omitempty"`
	Name                *string     `url:"name,omitempty" json:"name,omitempty"`
	Note                *string     `url:"note,omitempty" json:"note,omitempty"`
	Organization        *string     `url:"organization,omitempty" json:"organization,omitempty"`
	Password            *string     `url:"password,omitempty" json:"password,omitempty"`
	PrivateProfile      *bool       `url:"private_profile,omitempty" json:"private_profile,omitempty"`
	ProjectsLimit       *int        `url:"projects_limit,omitempty" json:"projects_limit,omitempty"`
	Provider            *string     `url:"provider,omitempty" json:"provider,omitempty"`
	ResetPassword       *bool       `url:"reset_password,omitempty" json:"reset_password,omitempty"`
	SkipConfirmation    *bool       `url:"skip_confirmation,omitempty" json:"skip_confirmation,omitempty"`
	Skype               *string     `url:"skype,omitempty" json:"skype,omitempty"`
	ThemeID             *int        `url:"theme_id,omitempty" json:"theme_id,omitempty"`
	Twitter             *string     `url:"twitter,omitempty" json:"twitter,omitempty"`
	Username            *string     `url:"username,omitempty" json:"username,omitempty"`
	WebsiteURL          *string     `url:"website_url,omitempty" json:"website_url,omitempty"`
}

// CreateUser creates a new user. Note only administrators can create new users.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#create-a-user
func (s *UsersService) CreateUser(opt *CreateUserOptions, options ...RequestOptionFunc) (*User, *Response, error) {
	var err error
	var req *retryablehttp.Request

	if opt.Avatar == nil {
		req, err = s.client.NewRequest(http.MethodPost, "users", opt, options)
	} else {
		req, err = s.client.UploadRequest(
			http.MethodPost,
			"users",
			opt.Avatar.Image,
			opt.Avatar.Filename,
			UploadAvatar,
			opt,
			options,
		)
	}
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, nil
}

// ModifyUserOptions represents the available ModifyUser() options.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#modify-a-user
type ModifyUserOptions struct {
	Admin              *bool       `url:"admin,omitempty" json:"admin,omitempty"`
	Avatar             *UserAvatar `url:"-" json:"avatar,omitempty"`
	Bio                *string     `url:"bio,omitempty" json:"bio,omitempty"`
	CanCreateGroup     *bool       `url:"can_create_group,omitempty" json:"can_create_group,omitempty"`
	CommitEmail        *string     `url:"commit_email,omitempty" json:"commit_email,omitempty"`
	Email              *string     `url:"email,omitempty" json:"email,omitempty"`
	External           *bool       `url:"external,omitempty" json:"external,omitempty"`
	ExternUID          *string     `url:"extern_uid,omitempty" json:"extern_uid,omitempty"`
	JobTitle           *string     `url:"job_title,omitempty" json:"job_title,omitempty"`
	Linkedin           *string     `url:"linkedin,omitempty" json:"linkedin,omitempty"`
	Location           *string     `url:"location,omitempty" json:"location,omitempty"`
	Name               *string     `url:"name,omitempty" json:"name,omitempty"`
	Note               *string     `url:"note,omitempty" json:"note,omitempty"`
	Organization       *string     `url:"organization,omitempty" json:"organization,omitempty"`
	Password           *string     `url:"password,omitempty" json:"password,omitempty"`
	PrivateProfile     *bool       `url:"private_profile,omitempty" json:"private_profile,omitempty"`
	ProjectsLimit      *int        `url:"projects_limit,omitempty" json:"projects_limit,omitempty"`
	Provider           *string     `url:"provider,omitempty" json:"provider,omitempty"`
	PublicEmail        *string     `url:"public_email,omitempty" json:"public_email,omitempty"`
	SkipReconfirmation *bool       `url:"skip_reconfirmation,omitempty" json:"skip_reconfirmation,omitempty"`
	Skype              *string     `url:"skype,omitempty" json:"skype,omitempty"`
	ThemeID            *int        `url:"theme_id,omitempty" json:"theme_id,omitempty"`
	Twitter            *string     `url:"twitter,omitempty" json:"twitter,omitempty"`
	Username           *string     `url:"username,omitempty" json:"username,omitempty"`
	WebsiteURL         *string     `url:"website_url,omitempty" json:"website_url,omitempty"`
}

// ModifyUser modifies an existing user. Only administrators can change attributes
// of a user.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#modify-a-user
func (s *UsersService) ModifyUser(user int, opt *ModifyUserOptions, options ...RequestOptionFunc) (*User, *Response, error) {
	var err error
	var req *retryablehttp.Request
	u := fmt.Sprintf("users/%d", user)

	if opt.Avatar == nil || (opt.Avatar.Filename == "" && opt.Avatar.Image == nil) {
		req, err = s.client.NewRequest(http.MethodPut, u, opt, options)
	} else {
		req, err = s.client.UploadRequest(
			http.MethodPut,
			u,
			opt.Avatar.Image,
			opt.Avatar.Filename,
			UploadAvatar,
			opt,
			options,
		)
	}
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, nil
}

// DeleteUser deletes a user. Available only for administrators. This is an
// idempotent function, calling this function for a non-existent user id still
// returns a status code 200 OK. The JSON response differs if the user was
// actually deleted or not. In the former the user is returned and in the
// latter not.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#delete-a-user
func (s *UsersService) DeleteUser(user int, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("users/%d", user)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// CurrentUser gets currently authenticated user.
//
// GitLab API docs: https://docs.gitlab.com/api/users/#get-the-current-user
func (s *UsersService) CurrentUser(options ...RequestOptionFunc) (*User, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "user", nil, options)
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, nil
}

// UserStatus represents the current status of a user
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#get-your-user-status
type UserStatus struct {
	Emoji         string            `json:"emoji"`
	Availability  AvailabilityValue `json:"availability"`
	Message       string            `json:"message"`
	MessageHTML   string            `json:"message_html"`
	ClearStatusAt *time.Time        `json:"clear_status_at"`
}

// CurrentUserStatus retrieves the user status
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#get-your-user-status
func (s *UsersService) CurrentUserStatus(options ...RequestOptionFunc) (*UserStatus, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "user/status", nil, options)
	if err != nil {
		return nil, nil, err
	}

	status := new(UserStatus)
	resp, err := s.client.Do(req, status)
	if err != nil {
		return nil, resp, err
	}

	return status, resp, nil
}

// GetUserStatus retrieves a user's status.
//
// uid can be either a user ID (int) or a username (string); will trim one "@" character off the username, if present.
// Other types will cause an error to be returned.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#get-the-status-of-a-user
func (s *UsersService) GetUserStatus(uid any, options ...RequestOptionFunc) (*UserStatus, *Response, error) {
	user, err := parseID(uid)
	if err != nil {
		return nil, nil, err
	}

	u := fmt.Sprintf("users/%s/status", strings.TrimPrefix(user, "@"))

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	status := new(UserStatus)
	resp, err := s.client.Do(req, status)
	if err != nil {
		return nil, resp, err
	}

	return status, resp, nil
}

// UserStatusOptions represents the options required to set the status
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#set-your-user-status
type UserStatusOptions struct {
	Emoji            *string                `url:"emoji,omitempty" json:"emoji,omitempty"`
	Availability     *AvailabilityValue     `url:"availability,omitempty" json:"availability,omitempty"`
	Message          *string                `url:"message,omitempty" json:"message,omitempty"`
	ClearStatusAfter *ClearStatusAfterValue `url:"clear_status_after,omitempty" json:"clear_status_after,omitempty"`
}

// SetUserStatus sets the user's status
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#set-your-user-status
func (s *UsersService) SetUserStatus(opt *UserStatusOptions, options ...RequestOptionFunc) (*UserStatus, *Response, error) {
	req, err := s.client.NewRequest(http.MethodPut, "user/status", opt, options)
	if err != nil {
		return nil, nil, err
	}

	status := new(UserStatus)
	resp, err := s.client.Do(req, status)
	if err != nil {
		return nil, resp, err
	}

	return status, resp, nil
}

// UserAssociationsCount represents the user associations count.
//
// Gitlab API docs:
// https://docs.gitlab.com/api/users/#get-a-count-of-a-users-projects-groups-issues-and-merge-requests
type UserAssociationsCount struct {
	GroupsCount        int `json:"groups_count"`
	ProjectsCount      int `json:"projects_count"`
	IssuesCount        int `json:"issues_count"`
	MergeRequestsCount int `json:"merge_requests_count"`
}

// GetUserAssociationsCount gets a list of a specified user associations.
//
// Gitlab API docs:
// https://docs.gitlab.com/api/users/#get-a-count-of-a-users-projects-groups-issues-and-merge-requests
func (s *UsersService) GetUserAssociationsCount(user int, options ...RequestOptionFunc) (*UserAssociationsCount, *Response, error) {
	u := fmt.Sprintf("users/%d/associations_count", user)

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	uac := new(UserAssociationsCount)
	resp, err := s.client.Do(req, uac)
	if err != nil {
		return nil, resp, err
	}

	return uac, resp, nil
}

// SSHKey represents a SSH key.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#list-all-ssh-keys
type SSHKey struct {
	ID        int        `json:"id"`
	Title     string     `json:"title"`
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	UsageType string     `json:"usage_type"`
}

// ListSSHKeysOptions represents the available ListSSHKeys options.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#list-all-ssh-keys
type ListSSHKeysOptions ListOptions

// ListSSHKeys gets a list of currently authenticated user's SSH keys.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#list-all-ssh-keys
func (s *UsersService) ListSSHKeys(opt *ListSSHKeysOptions, options ...RequestOptionFunc) ([]*SSHKey, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "user/keys", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var k []*SSHKey
	resp, err := s.client.Do(req, &k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// ListSSHKeysForUserOptions represents the available ListSSHKeysForUser() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_keys/#list-all-ssh-keys-for-a-user
type ListSSHKeysForUserOptions ListOptions

// ListSSHKeysForUser gets a list of a specified user's SSH keys.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_keys/#list-all-ssh-keys-for-a-user
func (s *UsersService) ListSSHKeysForUser(uid any, opt *ListSSHKeysForUserOptions, options ...RequestOptionFunc) ([]*SSHKey, *Response, error) {
	user, err := parseID(uid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("users/%s/keys", user)

	req, err := s.client.NewRequest(http.MethodGet, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var k []*SSHKey
	resp, err := s.client.Do(req, &k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// GetSSHKey gets a single key.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#get-an-ssh-key
func (s *UsersService) GetSSHKey(key int, options ...RequestOptionFunc) (*SSHKey, *Response, error) {
	u := fmt.Sprintf("user/keys/%d", key)

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(SSHKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// GetSSHKeyForUser gets a single key for a given user.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#get-an-ssh-key-for-a-user
func (s *UsersService) GetSSHKeyForUser(user int, key int, options ...RequestOptionFunc) (*SSHKey, *Response, error) {
	u := fmt.Sprintf("users/%d/keys/%d", user, key)

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(SSHKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// AddSSHKeyOptions represents the available AddSSHKey() options.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#add-an-ssh-key
type AddSSHKeyOptions struct {
	Title     *string  `url:"title,omitempty" json:"title,omitempty"`
	Key       *string  `url:"key,omitempty" json:"key,omitempty"`
	ExpiresAt *ISOTime `url:"expires_at,omitempty" json:"expires_at,omitempty"`
	UsageType *string  `url:"usage_type,omitempty" json:"usage_type,omitempty"`
}

// AddSSHKey creates a new key owned by the currently authenticated user.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#add-an-ssh-key
func (s *UsersService) AddSSHKey(opt *AddSSHKeyOptions, options ...RequestOptionFunc) (*SSHKey, *Response, error) {
	req, err := s.client.NewRequest(http.MethodPost, "user/keys", opt, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(SSHKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// AddSSHKeyForUser creates new key owned by specified user. Available only for
// admin.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#add-an-ssh-key-for-a-user
func (s *UsersService) AddSSHKeyForUser(user int, opt *AddSSHKeyOptions, options ...RequestOptionFunc) (*SSHKey, *Response, error) {
	u := fmt.Sprintf("users/%d/keys", user)

	req, err := s.client.NewRequest(http.MethodPost, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(SSHKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// DeleteSSHKey deletes key owned by currently authenticated user. This is an
// idempotent function and calling it on a key that is already deleted or not
// available results in 200 OK.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_keys/#delete-an-ssh-key
func (s *UsersService) DeleteSSHKey(key int, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("user/keys/%d", key)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DeleteSSHKeyForUser deletes key owned by a specified user. Available only
// for admin.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_keys/#delete-an-ssh-key-for-a-user
func (s *UsersService) DeleteSSHKeyForUser(user, key int, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("users/%d/keys/%d", user, key)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// GPGKey represents a GPG key.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#list-all-gpg-keys
type GPGKey struct {
	ID        int        `json:"id"`
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at"`
}

// ListGPGKeys gets a list of currently authenticated user’s GPG keys.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#list-all-gpg-keys
func (s *UsersService) ListGPGKeys(options ...RequestOptionFunc) ([]*GPGKey, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "user/gpg_keys", nil, options)
	if err != nil {
		return nil, nil, err
	}

	var ks []*GPGKey
	resp, err := s.client.Do(req, &ks)
	if err != nil {
		return nil, resp, err
	}

	return ks, resp, nil
}

// GetGPGKey gets a specific GPG key of currently authenticated user.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#get-a-gpg-key
func (s *UsersService) GetGPGKey(key int, options ...RequestOptionFunc) (*GPGKey, *Response, error) {
	u := fmt.Sprintf("user/gpg_keys/%d", key)

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(GPGKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// AddGPGKeyOptions represents the available AddGPGKey() options.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#add-a-gpg-key
type AddGPGKeyOptions struct {
	Key *string `url:"key,omitempty" json:"key,omitempty"`
}

// AddGPGKey creates a new GPG key owned by the currently authenticated user.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#add-a-gpg-key
func (s *UsersService) AddGPGKey(opt *AddGPGKeyOptions, options ...RequestOptionFunc) (*GPGKey, *Response, error) {
	req, err := s.client.NewRequest(http.MethodPost, "user/gpg_keys", opt, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(GPGKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// DeleteGPGKey deletes a GPG key owned by currently authenticated user.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#delete-a-gpg-key
func (s *UsersService) DeleteGPGKey(key int, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("user/gpg_keys/%d", key)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// ListGPGKeysForUser gets a list of a specified user’s GPG keys.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_keys/#list-all-gpg-keys-for-a-user
func (s *UsersService) ListGPGKeysForUser(user int, options ...RequestOptionFunc) ([]*GPGKey, *Response, error) {
	u := fmt.Sprintf("users/%d/gpg_keys", user)

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var ks []*GPGKey
	resp, err := s.client.Do(req, &ks)
	if err != nil {
		return nil, resp, err
	}

	return ks, resp, nil
}

// GetGPGKeyForUser gets a specific GPG key for a given user.
//
// GitLab API docs: https://docs.gitlab.com/api/user_keys/#get-a-gpg-key-for-a-user
func (s *UsersService) GetGPGKeyForUser(user, key int, options ...RequestOptionFunc) (*GPGKey, *Response, error) {
	u := fmt.Sprintf("users/%d/gpg_keys/%d", user, key)

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(GPGKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// AddGPGKeyForUser creates new GPG key owned by the specified user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_keys/#add-a-gpg-key-for-a-user
func (s *UsersService) AddGPGKeyForUser(user int, opt *AddGPGKeyOptions, options ...RequestOptionFunc) (*GPGKey, *Response, error) {
	u := fmt.Sprintf("users/%d/gpg_keys", user)

	req, err := s.client.NewRequest(http.MethodPost, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(GPGKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, nil
}

// DeleteGPGKeyForUser deletes a GPG key owned by a specified user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_keys/#delete-a-gpg-key-for-a-user
func (s *UsersService) DeleteGPGKeyForUser(user, key int, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("users/%d/gpg_keys/%d", user, key)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// Email represents an Email.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#list-all-email-addresses
type Email struct {
	ID          int        `json:"id"`
	Email       string     `json:"email"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
}

// ListEmails gets a list of currently authenticated user's Emails.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#list-all-email-addresses
func (s *UsersService) ListEmails(options ...RequestOptionFunc) ([]*Email, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "user/emails", nil, options)
	if err != nil {
		return nil, nil, err
	}

	var e []*Email
	resp, err := s.client.Do(req, &e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, nil
}

// ListEmailsForUserOptions represents the available ListEmailsForUser() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#list-all-email-addresses-for-a-user
type ListEmailsForUserOptions ListOptions

// ListEmailsForUser gets a list of a specified user's Emails. Available
// only for admin
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#list-all-email-addresses-for-a-user
func (s *UsersService) ListEmailsForUser(user int, opt *ListEmailsForUserOptions, options ...RequestOptionFunc) ([]*Email, *Response, error) {
	u := fmt.Sprintf("users/%d/emails", user)

	req, err := s.client.NewRequest(http.MethodGet, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var e []*Email
	resp, err := s.client.Do(req, &e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, nil
}

// GetEmail gets a single email.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#get-details-on-an-email-address
func (s *UsersService) GetEmail(email int, options ...RequestOptionFunc) (*Email, *Response, error) {
	u := fmt.Sprintf("user/emails/%d", email)

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	e := new(Email)
	resp, err := s.client.Do(req, e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, nil
}

// AddEmailOptions represents the available AddEmail() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#add-an-email-address
type AddEmailOptions struct {
	Email            *string `url:"email,omitempty" json:"email,omitempty"`
	SkipConfirmation *bool   `url:"skip_confirmation,omitempty" json:"skip_confirmation,omitempty"`
}

// AddEmail creates a new email owned by the currently authenticated user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#add-an-email-address
func (s *UsersService) AddEmail(opt *AddEmailOptions, options ...RequestOptionFunc) (*Email, *Response, error) {
	req, err := s.client.NewRequest(http.MethodPost, "user/emails", opt, options)
	if err != nil {
		return nil, nil, err
	}

	e := new(Email)
	resp, err := s.client.Do(req, e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, nil
}

// AddEmailForUser creates new email owned by specified user. Available only for
// admin.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#add-an-email-address-for-a-user
func (s *UsersService) AddEmailForUser(user int, opt *AddEmailOptions, options ...RequestOptionFunc) (*Email, *Response, error) {
	u := fmt.Sprintf("users/%d/emails", user)

	req, err := s.client.NewRequest(http.MethodPost, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	e := new(Email)
	resp, err := s.client.Do(req, e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, nil
}

// DeleteEmail deletes email owned by currently authenticated user. This is an
// idempotent function and calling it on a key that is already deleted or not
// available results in 200 OK.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#delete-an-email-address
func (s *UsersService) DeleteEmail(email int, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("user/emails/%d", email)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DeleteEmailForUser deletes email owned by a specified user. Available only
// for admin.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_email_addresses/#delete-an-email-address-for-a-user
func (s *UsersService) DeleteEmailForUser(user, email int, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("users/%d/emails/%d", user, email)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// BlockUser blocks the specified user. Available only for admin.
//
// GitLab API docs: https://docs.gitlab.com/api/user_moderation/#block-access-to-a-user
func (s *UsersService) BlockUser(user int, options ...RequestOptionFunc) error {
	u := fmt.Sprintf("users/%d/block", user)

	req, err := s.client.NewRequest(http.MethodPost, u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil && resp == nil {
		return err
	}

	switch resp.StatusCode {
	case 201:
		return nil
	case 403:
		return ErrUserBlockPrevented
	case 404:
		return ErrUserNotFound
	default:
		return fmt.Errorf("received unexpected result code: %d", resp.StatusCode)
	}
}

// UnblockUser unblocks the specified user. Available only for admin.
//
// GitLab API docs: https://docs.gitlab.com/api/user_moderation/#unblock-access-to-a-user
func (s *UsersService) UnblockUser(user int, options ...RequestOptionFunc) error {
	u := fmt.Sprintf("users/%d/unblock", user)

	req, err := s.client.NewRequest(http.MethodPost, u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil && resp == nil {
		return err
	}

	switch resp.StatusCode {
	case 201:
		return nil
	case 403:
		return ErrUserUnblockPrevented
	case 404:
		return ErrUserNotFound
	default:
		return fmt.Errorf("received unexpected result code: %d", resp.StatusCode)
	}
}

// BanUser bans the specified user. Available only for admin.
//
// GitLab API docs: https://docs.gitlab.com/api/user_moderation/#ban-a-user
func (s *UsersService) BanUser(user int, options ...RequestOptionFunc) error {
	u := fmt.Sprintf("users/%d/ban", user)

	req, err := s.client.NewRequest(http.MethodPost, u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil && resp == nil {
		return err
	}

	switch resp.StatusCode {
	case 201:
		return nil
	case 404:
		return ErrUserNotFound
	default:
		return fmt.Errorf("received unexpected result code: %d", resp.StatusCode)
	}
}

// UnbanUser unbans the specified user. Available only for admin.
//
// GitLab API docs: https://docs.gitlab.com/api/user_moderation/#unban-a-user
func (s *UsersService) UnbanUser(user int, options ...RequestOptionFunc) error {
	u := fmt.Sprintf("users/%d/unban", user)

	req, err := s.client.NewRequest(http.MethodPost, u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil && resp == nil {
		return err
	}

	switch resp.StatusCode {
	case 201:
		return nil
	case 404:
		return ErrUserNotFound
	default:
		return fmt.Errorf("received unexpected result code: %d", resp.StatusCode)
	}
}

// DeactivateUser deactivate the specified user. Available only for admin.
//
// GitLab API docs: https://docs.gitlab.com/api/user_moderation/#deactivate-a-user
func (s *UsersService) DeactivateUser(user int, options ...RequestOptionFunc) error {
	u := fmt.Sprintf("users/%d/deactivate", user)

	req, err := s.client.NewRequest(http.MethodPost, u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil && resp == nil {
		return err
	}

	switch resp.StatusCode {
	case 201:
		return nil
	case 403:
		return ErrUserDeactivatePrevented
	case 404:
		return ErrUserNotFound
	default:
		return fmt.Errorf("received unexpected result code: %d", resp.StatusCode)
	}
}

// ActivateUser activate the specified user. Available only for admin.
//
// GitLab API docs: https://docs.gitlab.com/api/user_moderation/#reactivate-a-user
func (s *UsersService) ActivateUser(user int, options ...RequestOptionFunc) error {
	u := fmt.Sprintf("users/%d/activate", user)

	req, err := s.client.NewRequest(http.MethodPost, u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil && resp == nil {
		return err
	}

	switch resp.StatusCode {
	case 201:
		return nil
	case 403:
		return ErrUserActivatePrevented
	case 404:
		return ErrUserNotFound
	default:
		return fmt.Errorf("received unexpected result code: %d", resp.StatusCode)
	}
}

// ApproveUser approve the specified user. Available only for admin.
//
// GitLab API docs: https://docs.gitlab.com/api/user_moderation/#approve-access-to-a-user
func (s *UsersService) ApproveUser(user int, options ...RequestOptionFunc) error {
	u := fmt.Sprintf("users/%d/approve", user)

	req, err := s.client.NewRequest(http.MethodPost, u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil && resp == nil {
		return err
	}

	switch resp.StatusCode {
	case 201:
		return nil
	case 403:
		return ErrUserApprovePrevented
	case 404:
		return ErrUserNotFound
	default:
		return fmt.Errorf("received unexpected result code: %d", resp.StatusCode)
	}
}

// RejectUser reject the specified user. Available only for admin.
//
// GitLab API docs: https://docs.gitlab.com/api/user_moderation/#reject-access-to-a-user
func (s *UsersService) RejectUser(user int, options ...RequestOptionFunc) error {
	u := fmt.Sprintf("users/%d/reject", user)

	req, err := s.client.NewRequest(http.MethodPost, u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil && resp == nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		return nil
	case 403:
		return ErrUserRejectPrevented
	case 404:
		return ErrUserNotFound
	case 409:
		return ErrUserConflict
	default:
		return fmt.Errorf("received unexpected result code: %d", resp.StatusCode)
	}
}

// ImpersonationToken represents an impersonation token.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#list-all-impersonation-tokens-for-a-user
type ImpersonationToken struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Active     bool       `json:"active"`
	Token      string     `json:"token"`
	Scopes     []string   `json:"scopes"`
	Revoked    bool       `json:"revoked"`
	CreatedAt  *time.Time `json:"created_at"`
	ExpiresAt  *ISOTime   `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// GetAllImpersonationTokensOptions represents the available
// GetAllImpersonationTokens() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#list-all-impersonation-tokens-for-a-user
type GetAllImpersonationTokensOptions struct {
	ListOptions
	State *string `url:"state,omitempty" json:"state,omitempty"`
}

// GetAllImpersonationTokens retrieves all impersonation tokens of a user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#list-all-impersonation-tokens-for-a-user
func (s *UsersService) GetAllImpersonationTokens(user int, opt *GetAllImpersonationTokensOptions, options ...RequestOptionFunc) ([]*ImpersonationToken, *Response, error) {
	u := fmt.Sprintf("users/%d/impersonation_tokens", user)

	req, err := s.client.NewRequest(http.MethodGet, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var ts []*ImpersonationToken
	resp, err := s.client.Do(req, &ts)
	if err != nil {
		return nil, resp, err
	}

	return ts, resp, nil
}

// GetImpersonationToken retrieves an impersonation token of a user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#get-an-impersonation-token-for-a-user
func (s *UsersService) GetImpersonationToken(user, token int, options ...RequestOptionFunc) (*ImpersonationToken, *Response, error) {
	u := fmt.Sprintf("users/%d/impersonation_tokens/%d", user, token)

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(ImpersonationToken)
	resp, err := s.client.Do(req, &t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, nil
}

// CreateImpersonationTokenOptions represents the available
// CreateImpersonationToken() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#create-an-impersonation-token
type CreateImpersonationTokenOptions struct {
	Name      *string    `url:"name,omitempty" json:"name,omitempty"`
	Scopes    *[]string  `url:"scopes,omitempty" json:"scopes,omitempty"`
	ExpiresAt *time.Time `url:"expires_at,omitempty" json:"expires_at,omitempty"`
}

// CreateImpersonationToken creates an impersonation token.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#create-an-impersonation-token
func (s *UsersService) CreateImpersonationToken(user int, opt *CreateImpersonationTokenOptions, options ...RequestOptionFunc) (*ImpersonationToken, *Response, error) {
	u := fmt.Sprintf("users/%d/impersonation_tokens", user)

	req, err := s.client.NewRequest(http.MethodPost, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(ImpersonationToken)
	resp, err := s.client.Do(req, &t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, nil
}

// RevokeImpersonationToken revokes an impersonation token.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#revoke-an-impersonation-token
func (s *UsersService) RevokeImpersonationToken(user, token int, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("users/%d/impersonation_tokens/%d", user, token)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// CreatePersonalAccessTokenOptions represents the available
// CreatePersonalAccessToken() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#create-a-personal-access-token-for-a-user
type CreatePersonalAccessTokenOptions struct {
	Name        *string   `url:"name,omitempty" json:"name,omitempty"`
	Description *string   `url:"description,omitempty" json:"description,omitempty"`
	ExpiresAt   *ISOTime  `url:"expires_at,omitempty" json:"expires_at,omitempty"`
	Scopes      *[]string `url:"scopes,omitempty" json:"scopes,omitempty"`
}

// CreatePersonalAccessToken creates a personal access token.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#create-a-personal-access-token-for-a-user
func (s *UsersService) CreatePersonalAccessToken(user int, opt *CreatePersonalAccessTokenOptions, options ...RequestOptionFunc) (*PersonalAccessToken, *Response, error) {
	u := fmt.Sprintf("users/%d/personal_access_tokens", user)

	req, err := s.client.NewRequest(http.MethodPost, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(PersonalAccessToken)
	resp, err := s.client.Do(req, &t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, nil
}

// CreatePersonalAccessTokenForCurrentUserOptions represents the available
// CreatePersonalAccessTokenForCurrentUser() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#create-a-personal-access-token
type CreatePersonalAccessTokenForCurrentUserOptions struct {
	Name        *string   `url:"name,omitempty" json:"name,omitempty"`
	Description *string   `url:"description,omitempty" json:"description,omitempty"`
	Scopes      *[]string `url:"scopes,omitempty" json:"scopes,omitempty"`
	ExpiresAt   *ISOTime  `url:"expires_at,omitempty" json:"expires_at,omitempty"`
}

// CreatePersonalAccessTokenForCurrentUser creates a personal access token with limited scopes for the currently authenticated user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_tokens/#create-a-personal-access-token
func (s *UsersService) CreatePersonalAccessTokenForCurrentUser(opt *CreatePersonalAccessTokenForCurrentUserOptions, options ...RequestOptionFunc) (*PersonalAccessToken, *Response, error) {
	u := "user/personal_access_tokens"

	req, err := s.client.NewRequest(http.MethodPost, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(PersonalAccessToken)
	resp, err := s.client.Do(req, &t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, nil
}

// UserActivity represents an entry in the user/activities response
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#list-a-users-activity
type UserActivity struct {
	Username       string   `json:"username"`
	LastActivityOn *ISOTime `json:"last_activity_on"`
}

// GetUserActivitiesOptions represents the options for GetUserActivities
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#list-a-users-activity
type GetUserActivitiesOptions struct {
	ListOptions
	From *ISOTime `url:"from,omitempty" json:"from,omitempty"`
}

// GetUserActivities retrieves user activities (admin only)
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#list-a-users-activity
func (s *UsersService) GetUserActivities(opt *GetUserActivitiesOptions, options ...RequestOptionFunc) ([]*UserActivity, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "user/activities", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var t []*UserActivity
	resp, err := s.client.Do(req, &t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, nil
}

// UserMembership represents a membership of the user in a namespace or project.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#list-projects-and-groups-that-a-user-is-a-member-of
type UserMembership struct {
	SourceID    int              `json:"source_id"`
	SourceName  string           `json:"source_name"`
	SourceType  string           `json:"source_type"`
	AccessLevel AccessLevelValue `json:"access_level"`
}

// GetUserMembershipOptions represents the options available to query user memberships.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#list-projects-and-groups-that-a-user-is-a-member-of
type GetUserMembershipOptions struct {
	ListOptions
	Type *string `url:"type,omitempty" json:"type,omitempty"`
}

// GetUserMemberships retrieves a list of the user's memberships.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#list-projects-and-groups-that-a-user-is-a-member-of
func (s *UsersService) GetUserMemberships(user int, opt *GetUserMembershipOptions, options ...RequestOptionFunc) ([]*UserMembership, *Response, error) {
	u := fmt.Sprintf("users/%d/memberships", user)

	req, err := s.client.NewRequest(http.MethodGet, u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var m []*UserMembership
	resp, err := s.client.Do(req, &m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, nil
}

// DisableTwoFactor disables two factor authentication for the specified user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#disable-two-factor-authentication-for-a-user
func (s *UsersService) DisableTwoFactor(user int, options ...RequestOptionFunc) error {
	u := fmt.Sprintf("users/%d/disable_two_factor", user)

	req, err := s.client.NewRequest(http.MethodPatch, u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil && resp == nil {
		return err
	}

	switch resp.StatusCode {
	case 204:
		return nil
	case 400:
		return ErrUserTwoFactorNotEnabled
	case 403:
		return ErrUserDisableTwoFactorPrevented
	case 404:
		return ErrUserNotFound
	default:
		return fmt.Errorf("received unexpected result code: %d", resp.StatusCode)
	}
}

// UserRunner represents a GitLab runner linked to the current user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#create-a-runner-linked-to-a-user
type UserRunner struct {
	ID             int        `json:"id"`
	Token          string     `json:"token"`
	TokenExpiresAt *time.Time `json:"token_expires_at"`
}

// CreateUserRunnerOptions represents the available CreateUserRunner() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#create-a-runner-linked-to-a-user
type CreateUserRunnerOptions struct {
	RunnerType      *string   `url:"runner_type,omitempty" json:"runner_type,omitempty"`
	GroupID         *int      `url:"group_id,omitempty" json:"group_id,omitempty"`
	ProjectID       *int      `url:"project_id,omitempty" json:"project_id,omitempty"`
	Description     *string   `url:"description,omitempty" json:"description,omitempty"`
	Paused          *bool     `url:"paused,omitempty" json:"paused,omitempty"`
	Locked          *bool     `url:"locked,omitempty" json:"locked,omitempty"`
	RunUntagged     *bool     `url:"run_untagged,omitempty" json:"run_untagged,omitempty"`
	TagList         *[]string `url:"tag_list,omitempty" json:"tag_list,omitempty"`
	AccessLevel     *string   `url:"access_level,omitempty" json:"access_level,omitempty"`
	MaximumTimeout  *int      `url:"maximum_timeout,omitempty" json:"maximum_timeout,omitempty"`
	MaintenanceNote *string   `url:"maintenance_note,omitempty" json:"maintenance_note,omitempty"`
}

// CreateUserRunner creates a runner linked to the current user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#create-a-runner-linked-to-a-user
func (s *UsersService) CreateUserRunner(opts *CreateUserRunnerOptions, options ...RequestOptionFunc) (*UserRunner, *Response, error) {
	req, err := s.client.NewRequest(http.MethodPost, "user/runners", opts, options)
	if err != nil {
		return nil, nil, err
	}

	r := new(UserRunner)
	resp, err := s.client.Do(req, r)
	if err != nil {
		return nil, resp, err
	}

	return r, resp, nil
}

// CreateServiceAccountUserOptions represents the available CreateServiceAccountUser() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_service_accounts/#create-a-service-account-user
type CreateServiceAccountUserOptions struct {
	Name     *string `url:"name,omitempty" json:"name,omitempty"`
	Username *string `url:"username,omitempty" json:"username,omitempty"`
	Email    *string `url:"email,omitempty" json:"email,omitempty"`
}

// CreateServiceAccountUser creates a new service account user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_service_accounts/#create-a-service-account-user
func (s *UsersService) CreateServiceAccountUser(opts *CreateServiceAccountUserOptions, options ...RequestOptionFunc) (*User, *Response, error) {
	req, err := s.client.NewRequest(http.MethodPost, "service_accounts", opts, options)
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, nil
}

// ListServiceAccounts lists all service accounts.
//
// GitLab API docs:
// https://docs.gitlab.com/api/user_service_accounts/#list-all-service-account-users
func (s *UsersService) ListServiceAccounts(opt *ListServiceAccountsOptions, options ...RequestOptionFunc) ([]*ServiceAccount, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "service_accounts", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var sas []*ServiceAccount
	resp, err := s.client.Do(req, &sas)
	if err != nil {
		return nil, resp, err
	}

	return sas, resp, nil
}

// UploadAvatar uploads an avatar to the current user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#upload-an-avatar-for-yourself
func (s *UsersService) UploadAvatar(avatar io.Reader, filename string, options ...RequestOptionFunc) (*User, *Response, error) {
	u := "user/avatar"

	req, err := s.client.UploadRequest(
		http.MethodPut,
		u,
		avatar,
		filename,
		UploadAvatar,
		nil,
		options,
	)
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, nil
}

// DeleteUserIdentity deletes a user's authentication identity using the provider
// name associated with that identity. Only available for administrators.
//
// GitLab API docs:
// https://docs.gitlab.com/api/users/#delete-authentication-identity-from-a-user
func (s *UsersService) DeleteUserIdentity(user int, provider string, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("users/%d/identities/%s", user, provider)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

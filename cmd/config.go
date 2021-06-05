package main

/*
 * AWS SSO CLI
 * Copyright (c) 2021 Aaron Turner  <synfinatic at gmail dot com>
 *
 * This program is free software: you can redistribute it
 * and/or modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or with the authors permission any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

import (
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type AWSProfile struct {
	Alias     string `koanf:"Alias" yaml:"Alias,omitempty"`         // Friendly name
	Role      string `koanf:"Role" yaml:"Role,omitempty"`           // AWS Role Name
	Region    string `koanf:"Region" yaml:"Region,omitempty"`       // AWS Default Region
	AccountId string `koanf:"AccountId" yaml:"AccountId,omitempty"` // AWS AccountId
}

type ConfigFile struct {
	SSO          map[string]*SSOConfig `koanf:"SSOConfig" yaml:"SSOConfig,omitempty"`
	DefaultSSO   string                `koanf:"DefaultSSO" yaml:"DefaultSSO,omitempty"`   // specify default SSO by key
	SecureStore  string                `koanf:"SecureStore" yaml:"SecureStore,omitempty"` // json or keyring
	JsonStore    JsonStoreConfig       `koanf:"Json" yaml:"Json,omitempty"`
	KeyringStore KeyringStoreConfig    `koanf:"Keyring" yaml:"Keyring,omitempty"`
}

type SSOConfig struct {
	SSORegion     string                `koanf:"SSORegion" yaml:"SSORegion"`
	StartUrl      string                `koanf:"StartUrl" yaml:"StartUrl"`
	Accounts      map[int64]*SSOAccount `koanf:"Accounts" yaml:"Accounts,omitempty"`
	DefaultRegion string                `koanf:"DefaultRegion" yaml:"DefaultRegion,omitempty"`
}

type SSOAccount struct {
	Name          string            `koanf:"Name" yaml:"Name,omitempty"` // Admin configured Account Name
	Tags          map[string]string `koanf:"Tags" yaml:"Tags,omitempty" `
	Roles         []*SSORole        `koanf:"Roles" yaml:"Roles,omitempty"`
	DefaultRegion string            `koanf:"DefaultRegion" yaml:"DefaultRegion,omitempty"`
}

type SSORole struct {
	account       *SSOAccount
	ARN           string            `koanf:"ARN" yaml:"ARN"`
	Profile       string            `koanf:"Profile" yaml:"Profile,omitempty"`
	Tags          map[string]string `koanf:"Tags" yaml:"Tags,omitempty"`
	DefaultRegion string            `koanf:"DefaultRegion" yaml:"DefaultRegion,omitempty"`
}

type JsonStoreConfig struct {
	File string `koanf:"File" yaml:"File"` // Filename
}

type KeyringStoreConfig struct {
	// ???
}

// Refresh should be called any time you load the SSOConfig into memory or add a role
// to update the Role -> Account references
func (s *SSOConfig) Refresh() {
	for _, a := range s.Accounts {
		for _, r := range a.Roles {
			r.account = a
		}
	}
}

func (a *SSOAccount) HasRole(arn string) bool {
	hasRole := false
	for _, role := range a.Roles {
		if role.ARN == arn {
			hasRole = true
			break
		}
	}
	return hasRole
}

func (s *SSOConfig) UpdateRoles(roles map[string][]RoleInfo) (int64, error) {
	var changes int64 = 0
	for account, accountInfo := range roles {
		accountId, err := strconv.ParseInt(account, 10, 64)
		if err != nil {
			return 0, err
		}
		if s.Accounts == nil {
			s.Accounts = map[int64]*SSOAccount{}
		}
		_, hasAccount := s.Accounts[accountId]
		if !hasAccount {
			s.Accounts[accountId] = &SSOAccount{
				Name:  accountInfo[0].AccountName,
				Roles: []*SSORole{},
			}
		}

		for _, roleInfo := range accountInfo {
			if !hasAccount || !s.Accounts[accountId].HasRole(roleInfo.RoleArn()) {
				changes += 1
				s.Accounts[accountId].Roles = append(s.Accounts[accountId].Roles, &SSORole{
					ARN:     roleInfo.RoleArn(),
					Profile: roleInfo.Profile,
				})
			}
		}
	}
	if changes > 0 {
		s.Refresh()
	}
	return changes, nil
}

// GetRoles returns a list of all the roles for this SSOConfig
func (s *SSOConfig) GetRoles() []*SSORole {
	roles := []*SSORole{}
	for _, a := range s.Accounts {
		for _, r := range a.Roles {
			roles = append(roles, r)
		}
	}
	return roles
}

// GetAllTags returns all of the user defined tags and calculated tags for this account
func (a *SSOAccount) GetAllTags(id int64) map[string]string {
	tags := map[string]string{
		"AccountName": a.Name,
	}
	if id > 0 {
		accountId := strconv.FormatInt(id, 10)
		tags["AccountId"] = accountId
	}
	if a.DefaultRegion != "" {
		tags["DefaultRegion"] = a.DefaultRegion
	}
	for k, v := range a.Tags {
		tags[k] = v
	}
	return tags
}

// GetAllTags returns all of the user defined and calculated tags for this role
func (r *SSORole) GetAllTags() map[string]string {
	tags := map[string]string{}
	// First pull in the account tags
	for k, v := range r.account.GetAllTags(r.GetAccountId64()) {
		tags[k] = v
	}

	// Then override/add any specific tags
	tags["RoleName"] = r.GetRoleName()
	tags["AccountId"] = r.GetAccountId()

	if r.DefaultRegion != "" {
		tags["DefaultRegion"] = r.DefaultRegion
	}
	for k, v := range r.Tags {
		tags[k] = v
	}

	return tags
}

// GetRoleName returns the role name portion of the ARN
func (r *SSORole) GetRoleName() string {
	s := strings.Split(r.ARN, "/")
	return s[1]
}

// GetAccountId returns the accountId portion of the ARN
func (r *SSORole) GetAccountId() string {
	s := strings.Split(r.ARN, ":")
	return s[3]
}

// GetAccountId64 returns the accountId portion of the ARN
func (r *SSORole) GetAccountId64() int64 {
	s := strings.Split(r.ARN, ":")
	i, err := strconv.ParseInt(s[3], 10, 64)
	if err != nil {
		log.WithError(err).Panicf("Unable to decode account id for %s", r.ARN)
	}
	return i
}

// insertSortedString inserts s into ss in a sorted manner
func insertSortedString(ss []string, s string) []string {
	i := sort.SearchStrings(ss, s)
	ss = append(ss, "")
	copy(ss[i+1:], ss[i:])
	ss[i] = s
	return ss
}

// addKeyValue inserts the v into the slice for the given k
func addKeyValue(tags *map[string][]string, k, v string) {
	t := *tags
	if t[k] == nil {
		t[k] = []string{}
	}
	hasValue := false
	for _, value := range t[k] {
		if value == v {
			hasValue = true
			break
		}
	}
	if !hasValue {
		t[k] = insertSortedString(t[k], v)
	}
}

// returns all of the available account & role tags for our SSO Provider
func (s *SSOConfig) GetAllTags() map[string][]string {
	tags := map[string][]string{}
	for account, accountInfo := range s.Accounts {
		if accountInfo.Tags != nil {
			for k, v := range accountInfo.GetAllTags(account) {
				addKeyValue(&tags, k, v)
			}
		}
		for _, roleInfo := range accountInfo.Roles {
			for k, v := range roleInfo.GetAllTags() {
				addKeyValue(&tags, k, v)
			}
		}
	}
	return tags
}

// GetRoleMatches finds all the roles which match all of the given tags
func (s *SSOConfig) GetRoleMatches(tags map[string]string) []*SSORole {
	allRoles := s.GetRoles()
	match := []*SSORole{}
	for _, role := range allRoles {
		hasMatch := make(map[string]bool, len(tags))
		for tk, tv := range tags {
			for k, v := range role.GetAllTags() {
				if k == tk && v == tv {
					hasMatch[tk] = true
					break
				}
			}
		}
		isMatch := true
		for _, v := range hasMatch {
			if v == false {
				isMatch = false
				break
			}
		}
		if isMatch {
			match = append(match, role)
		}

	}
	return match
}

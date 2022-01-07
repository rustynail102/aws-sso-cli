package sso

/*
 * AWS SSO CLI
 * Copyright (c) 2021-2022 Aaron Turner  <synfinatic at gmail dot com>
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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	// "github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/aws-sso-cli/utils"
)

const CACHE_VERSION = 3

type SSOCache struct {
	LastUpdate int64    `json:"LastUpdate,omitempty"` // when these records for this SSO were updated
	History    []string `json:"History,omitempty"`
	Roles      *Roles   `json:"Roles,omitempty"`
	name       string   // name of this SSO Instance
}

// Our Cachefile.  Sub-structs defined in sso/cache.go
type Cache struct {
	Version         int64                `json:"Version"`
	settings        *Settings            // pointer back up
	ConfigCreatedAt int64                `json:"ConfigCreatedAt"` // track config.yaml
	SSO             map[string]*SSOCache `json:"SSO,omitempty"`
	ssoName         string               // name of SSO that is active
}

func OpenCache(f string, s *Settings) (*Cache, error) {
	cache := Cache{
		settings:        s,
		ConfigCreatedAt: 0,
		Version:         1, // use an invalid default version for cache files without a version
		SSO:             map[string]*SSOCache{},
		ssoName:         s.DefaultSSO,
	}

	var err error
	var cacheBytes []byte
	if f != "" {
		cacheBytes, err = ioutil.ReadFile(f)
		if err != nil {
			return &cache, err // return empty struct
		}
		err = json.Unmarshal(cacheBytes, &cache)
	}

	c := &cache
	c.deleteOldHistory()

	return c, err
}

// GetSSO returns the current SSOCache object for the current SSO instance
func (c *Cache) GetSSO() *SSOCache {
	if v, ok := c.SSO[c.ssoName]; ok {
		v.name = c.ssoName
		v.Roles.ssoName = c.ssoName
		return v
	}

	// init a new one
	c.SSO[c.ssoName] = &SSOCache{
		name:       c.ssoName,
		LastUpdate: 0,
		History:    []string{},
		Roles: &Roles{
			Accounts: map[int64]*AWSAccount{},
			ssoName:  c.ssoName,
		},
	}
	return c.SSO[c.ssoName]
}

// Expired returns if our Roles cache data is too old.
// If configFile is a valid file, we check the lastModificationTime of that file
// vs. the ConfigCreatedAt to determine if the cache needs to be updated
func (c *Cache) Expired(s *SSOConfig) error {
	if c.Version < CACHE_VERSION {
		return fmt.Errorf("Local cache is out of date; current cache version %d is less than %d", c.Version, CACHE_VERSION)
	}

	cache := c.GetSSO()
	if cache.LastUpdate+CACHE_TTL < time.Now().Unix() {
		return fmt.Errorf("Local cache is out of date; TTL has been exceeded.")
	}

	if s.CreatedAt() > c.ConfigCreatedAt {
		return fmt.Errorf("Local cache is out of date; config.yaml modified.")
	}
	return nil
}

func (c *Cache) CacheFile() string {
	return c.settings.cacheFile
}

// Save saves our cache to the current file
func (c *Cache) Save(updateTime bool) error {
	c.Version = CACHE_VERSION
	if updateTime {
		cache := c.GetSSO()
		cache.LastUpdate = time.Now().Unix()
	}
	jbytes, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("Unable to masrhal json: %s", err.Error())
	}
	err = utils.EnsureDirExists(c.CacheFile())
	if err != nil {
		return fmt.Errorf("Unable to create directory for %s: %s", c.CacheFile(), err.Error())
	}
	err = ioutil.WriteFile(c.CacheFile(), jbytes, 0600)
	if err != nil {
		return fmt.Errorf("Unable to write %s: %s", c.CacheFile(), err.Error())
	}
	return nil
}

// adds a role to the History list up to the max number of entries
// and then removes the History tag from any roles that aren't in our list
func (c *Cache) AddHistory(item string) {
	cache := c.GetSSO()
	// If it's already in the list, remove it
	for x, h := range cache.History {
		if h == item {
			// delete from history
			cache.History = append(cache.History[:x], cache.History[x+1:]...)
			break
		}
	}

	cache.History = append([]string{item}, cache.History...) // push on top
	for int64(len(cache.History)) > c.settings.HistoryLimit {
		// remove the oldest entry
		cache.History = cache.History[:len(cache.History)-1]
	}

	// Update our Tags for this new item
	aId, roleName, _ := utils.ParseRoleARN(item)
	if a, ok := cache.Roles.Accounts[aId]; ok {
		if r, ok := a.Roles[roleName]; ok {
			r.Tags["History"] = fmt.Sprintf("%s:%s,%d", a.Alias, roleName, time.Now().Unix())
		}
	}

	// remove any history tags not in our list
	roles := cache.Roles.MatchingRolesWithTagKey("History")
	for _, role := range roles {
		exists := false
		for _, history := range cache.History {
			if history == (*role).Arn {
				exists = true
				break
			}
		}
		if !exists {
			aId, roleName, _ := utils.ParseRoleARN(role.Arn)
			delete(cache.Roles.Accounts[aId].Roles[roleName].Tags, "History")
		}
	}
}

// deleteOldHistory removes any items from history which are older than HistoryMinutes
// Does not actually save to disk, only updates in memory cache
func (c *Cache) deleteOldHistory() {
	if c.settings.HistoryMinutes <= 0 {
		// no op if HistoryMinutes <= 0
		return
	}

	cache := c.GetSSO()

	newHistoryItems := []string{}
	for _, arn := range cache.History {
		id, role, err := utils.ParseRoleARN(arn)
		if err != nil {
			log.Errorf("Unable to parse History ARN %s: %s", arn, err.Error())
			continue
		}

		if a, ok := cache.Roles.Accounts[id]; ok {
			if r, ok := a.Roles[role]; ok {
				// figure out if this history item has expired
				values := strings.SplitN(r.Tags["History"], ",", 2)
				if len(values) != 2 {
					log.Errorf("Too few fields for %s History Tag: '%s'", r.Arn, r.Tags["History"])
					continue
				}
				lastTime, err := strconv.ParseInt(values[1], 10, 64)
				if err != nil {
					log.Errorf("Unable to parse %s History Tag '%s': %s", r.Arn, r.Tags["History"], err.Error())
					continue
				}

				d := time.Since(time.Unix(lastTime, 0))
				if int64(d.Minutes()) < c.settings.HistoryMinutes {
					// keep current entries in our list
					newHistoryItems = append(newHistoryItems, arn)
				} else {
					// else, delete the tag
					delete(r.Tags, "History")
				}
			}
		}
	}

	cache.History = newHistoryItems
}

// Refresh updates our cached Roles based on AWS SSO & our Config
// but does not save this data!
func (c *Cache) Refresh(sso *AWSSSO, config *SSOConfig) error {
	r, err := c.NewRoles(sso, config)
	if err != nil {
		return err
	}
	cache := c.GetSSO()
	cache.Roles = r
	c.ConfigCreatedAt = config.CreatedAt()
	return nil
}

// Update the Expires time in the cache.  expires is Unix epoch time in sec
func (c *Cache) SetRoleExpires(arn string, expires int64) error {
	flat, err := c.GetRole(arn)
	if err != nil {
		return err
	}

	cache := c.GetSSO()
	cache.Roles.Accounts[flat.AccountId].Roles[flat.RoleName].Expires = expires
	return c.Save(false)
}

func (c *Cache) MarkRolesExpired() error {
	cache := c.GetSSO()
	for accountId := range cache.Roles.Accounts {
		for _, role := range cache.Roles.Accounts[accountId].Roles {
			(*role).Expires = 0
		}
	}
	return c.Save(false)
}

// returns all tags, but with with spaces replaced with underscores
func (c *Cache) GetAllTagsSelect() *TagsList {
	cache := c.GetSSO()
	tags := cache.Roles.GetAllTags()
	fixedTags := NewTagsList()
	for k, values := range *tags {
		key := strings.ReplaceAll(k, " ", "_")
		for _, v := range values {
			if key == "History" {
				v = reformatHistory(v)
			}
			fixedTags.Add(key, strings.ReplaceAll(v, " ", "_"))
		}
	}
	return fixedTags
}

// GetRoleTagsSelect returns all the tags for each role with all the spaces
// replaced with underscores
func (c *Cache) GetRoleTagsSelect() *RoleTags {
	ret := RoleTags{}
	cache := c.GetSSO()
	fList := cache.Roles.GetAllRoles()
	for _, role := range fList {
		ret[role.Arn] = map[string]string{}
		for k, v := range role.Tags {
			key := strings.ReplaceAll(k, " ", "_")
			if key == "History" {
				v = reformatHistory(v)
			}
			value := strings.ReplaceAll(v, " ", "_")
			ret[role.Arn][key] = value
		}
	}
	return &ret
}

// GetRole returns the AWSRoleFlat for the given role ARN
func (c *Cache) GetRole(arn string) (*AWSRoleFlat, error) {
	accountId, roleName, err := utils.ParseRoleARN(arn)
	if err != nil {
		return &AWSRoleFlat{}, err
	}
	cache := c.GetSSO()
	return cache.Roles.GetRole(accountId, roleName)
}

// Merges the AWS SSO and our Config file to create our Roles struct
// which is defined in cache_roles.go
func (c *Cache) NewRoles(as *AWSSSO, config *SSOConfig) (*Roles, error) {
	r := Roles{
		SSORegion:     config.SSORegion,
		StartUrl:      config.StartUrl,
		DefaultRegion: config.DefaultRegion,
		Accounts:      map[int64]*AWSAccount{},
		ssoName:       config.settings.DefaultSSO,
	}

	cache := c.GetSSO()

	// First go through all the AWS SSO Accounts & Roles
	accounts, err := as.GetAccounts()
	if err != nil {
		return &r, fmt.Errorf("Unable to get AWS SSO accounts: %s", err.Error())
	}
	for _, aInfo := range accounts {
		accountId := aInfo.GetAccountId64()
		r.Accounts[accountId] = &AWSAccount{
			Alias:        aInfo.AccountName, // AWS SSO calls it `AccountName`
			EmailAddress: aInfo.EmailAddress,
			Tags:         map[string]string{},
			Roles:        map[string]*AWSRole{},
		}

		roles, err := as.GetRoles(aInfo)
		if err != nil {
			return &r, fmt.Errorf("Unable to get AWS SSO roles: %s", err.Error())
		}
		for _, role := range roles {
			r.Accounts[accountId].Roles[role.RoleName] = &AWSRole{
				Arn: utils.MakeRoleARN(accountId, role.RoleName),
				Tags: map[string]string{
					"AccountID":    aInfo.AccountId,
					"AccountAlias": aInfo.AccountName, // AWS SSO calls it `AccountName`
					"Email":        aInfo.EmailAddress,
					"Role":         role.RoleName,
				},
			}
			// need to copy over the Expires & History fields from our current cache
			if _, ok := cache.Roles.Accounts[accountId]; ok {
				if _, ok := cache.Roles.Accounts[accountId].Roles[role.RoleName]; ok {
					if expires := cache.Roles.Accounts[accountId].Roles[role.RoleName].Expires; expires > 0 {
						r.Accounts[accountId].Roles[role.RoleName].Expires = expires
					}
					if v, ok := cache.Roles.Accounts[accountId].Roles[role.RoleName].Tags["History"]; ok {
						r.Accounts[accountId].Roles[role.RoleName].Tags["History"] = v
					}
				}
			}
		}
	}

	// The load all the Config file stuff.  Normally this is just adding markup, but
	// for accounts &roles that are not in SSO, we may be creating them as well!
	for accountId, account := range config.Accounts {
		if _, ok := r.Accounts[accountId]; !ok {
			r.Accounts[accountId] = &AWSAccount{
				Tags:  map[string]string{},
				Roles: map[string]*AWSRole{},
			}
		}
		r.Accounts[accountId].DefaultRegion = account.DefaultRegion
		r.Accounts[accountId].Name = account.Name

		// set our account tags
		for k, v := range config.Accounts[accountId].Tags {
			r.Accounts[accountId].Tags[k] = v
		}

		// set the AWS SSO tags for all the SSO roles
		for roleName := range r.Accounts[accountId].Roles {
			aId, _ := utils.AccountIdToString(accountId)
			r.Accounts[accountId].Roles[roleName].Tags["AccountID"] = aId
			r.Accounts[accountId].Roles[roleName].Tags["AccountAlias"] = r.Accounts[accountId].Alias
			r.Accounts[accountId].Roles[roleName].Tags["Email"] = r.Accounts[accountId].EmailAddress
			r.Accounts[accountId].Roles[roleName].Tags["Role"] = roleName
			if r.Accounts[accountId].Name != "" {
				r.Accounts[accountId].Roles[roleName].Tags["AccountName"] = r.Accounts[accountId].Name
			}
			if r.Accounts[accountId].Roles[roleName].DefaultRegion != "" {
				r.Accounts[accountId].Roles[roleName].Tags["DefaultRegion"] = r.Accounts[accountId].Roles[roleName].DefaultRegion
			}
		}

		// set the tags from the config file
		for roleName, role := range config.Accounts[accountId].Roles {
			if _, ok := r.Accounts[accountId].Roles[roleName]; !ok {
				r.Accounts[accountId].Roles[roleName] = &AWSRole{
					Tags: map[string]string{},
				}
			}
			r.Accounts[accountId].Roles[roleName].Arn = utils.MakeRoleARN(accountId, roleName)
			r.Accounts[accountId].Roles[roleName].Profile = role.Profile
			r.Accounts[accountId].Roles[roleName].DefaultRegion = r.Accounts[accountId].DefaultRegion
			r.Accounts[accountId].Roles[roleName].Via = role.Via
			if role.DefaultRegion != "" {
				r.Accounts[accountId].Roles[roleName].DefaultRegion = role.DefaultRegion
			}
			// Copy the account tags to the role
			for k, v := range config.Accounts[accountId].Tags {
				r.Accounts[accountId].Roles[roleName].Tags[k] = v
			}
			// Insert role specific tags (possible overwrite of account level)
			for k, v := range role.Tags {
				r.Accounts[accountId].Roles[roleName].Tags[k] = v
			}
		}
	}

	return &r, nil
}

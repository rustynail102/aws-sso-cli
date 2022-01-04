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
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/synfinatic/aws-sso-cli/storage"
)

const TEST_JSON_STORE_FILE = "../storage/testdata/store.json"

type CacheRolesTestSuite struct {
	suite.Suite
	cache     *Cache
	cacheFile string
	settings  *Settings
	storage   storage.SecureStorage
	jsonFile  string
}

func TestCacheRolesTestSuite(t *testing.T) {
	// copy our cache test file to a temp file
	f, err := os.CreateTemp("", "*")
	assert.NoError(t, err)
	f.Close()

	settings := &Settings{
		HistoryLimit:   1,
		HistoryMinutes: 90,
		DefaultSSO:     "Default",
		cacheFile:      f.Name(),
	}

	// cache
	input, err := ioutil.ReadFile(TEST_CACHE_FILE)
	assert.NoError(t, err)

	err = ioutil.WriteFile(f.Name(), input, 0600)
	assert.NoError(t, err)

	c, err := OpenCache(f.Name(), settings)
	assert.NoError(t, err)

	// secure store
	f2, err := os.CreateTemp("", "*")
	assert.Nil(t, err)

	jsonFile := f2.Name()
	f2.Close()

	input, err = ioutil.ReadFile(TEST_JSON_STORE_FILE)
	assert.Nil(t, err)

	err = ioutil.WriteFile(jsonFile, input, 0600)
	assert.Nil(t, err)

	sstore, err := storage.OpenJsonStore(jsonFile)
	assert.Nil(t, err)

	defaults := map[string]interface{}{}
	over := OverrideSettings{}
	set, err := LoadSettings(TEST_SETTINGS_FILE, TEST_CACHE_FILE, defaults, over)
	assert.NoError(t, err)

	s := &CacheRolesTestSuite{
		cache:     c,
		cacheFile: f.Name(),
		settings:  set,
		storage:   sstore,
		jsonFile:  jsonFile,
	}
	suite.Run(t, s)
}

func (suite *CacheRolesTestSuite) TearDownAllSuite() {
	os.Remove(suite.cacheFile)
	os.Remove(suite.jsonFile)
}

func (suite *CacheRolesTestSuite) TestAccountIds() {
	t := suite.T()
	roles := suite.cache.SSO[suite.cache.ssoName].Roles

	assert.NotEmpty(t, roles.AccountIds())
	assert.Contains(t, roles.AccountIds(), int64(258234615182))
	assert.NotContains(t, roles.AccountIds(), int64(2582346))
}

func (suite *CacheRolesTestSuite) TestGetAllRoles() {
	t := suite.T()

	roles := suite.cache.SSO[suite.cache.ssoName].Roles
	flat := roles.GetAllRoles()
	assert.NotEmpty(t, flat)
}

func (suite *CacheRolesTestSuite) TestGetAccountRoles() {
	t := suite.T()
	roles := suite.cache.SSO[suite.cache.ssoName].Roles

	flat := roles.GetAccountRoles(258234615182)
	assert.NotEmpty(t, flat)

	flat = roles.GetAccountRoles(258234615)
	assert.Empty(t, flat)
}

func (suite *CacheRolesTestSuite) TestGetAllTags() {
	t := suite.T()
	roles := suite.cache.SSO[suite.cache.ssoName].Roles

	tags := *(roles.GetAllTags())
	assert.NotEmpty(t, tags)
	assert.Contains(t, tags["Email"], "control-tower-dev-aws@ourcompany.com")
	assert.NotContains(t, tags["Email"], "foobar@ourcompany.com")
}

func (suite *CacheRolesTestSuite) TestGetRoleTags() {
	t := suite.T()
	roles := suite.cache.SSO[suite.cache.ssoName].Roles

	tags := *(roles.GetRoleTags())
	assert.NotEmpty(t, tags)
	arn := "arn:aws:iam::258234615182:role/AWSAdministratorAccess"
	assert.Contains(t, tags, arn)
	assert.NotContains(t, tags, "foobar")
	assert.Contains(t, tags[arn]["Email"], "control-tower-dev-aws@ourcompany.com")
	assert.NotContains(t, tags[arn]["Email"], "foobar@ourcompany.com")
}

func (suite *CacheRolesTestSuite) TestGetRole() {
	t := suite.T()
	roles := suite.cache.SSO[suite.cache.ssoName].Roles

	r, err := roles.GetRole(258234615182, "AWSAdministratorAccess")
	assert.NoError(t, err)
	assert.Equal(t, int64(258234615182), r.AccountId)
	assert.Equal(t, "AWSAdministratorAccess", r.RoleName)
	assert.Equal(t, "", r.Profile)
	p, err := r.ProfileName(suite.settings)
	assert.NoError(t, err)
	assert.Equal(t, "OurCompany Control Tower Playground/AWSAdministratorAccess", p)
}

func (suite *CacheRolesTestSuite) TestProfileName() {
	t := suite.T()
	roles := suite.cache.SSO[suite.cache.ssoName].Roles
	r, err := roles.GetRole(258234615182, "AWSAdministratorAccess")
	assert.NoError(t, err)

	p, err := r.ProfileName(suite.settings)
	assert.NoError(t, err)
	assert.Equal(t, "OurCompany Control Tower Playground/AWSAdministratorAccess", p)

	settings := suite.settings
	settings.ProfileFormat = `{{ FirstItem .AccountName .AccountAlias | StringReplace " " "_" }}:{{ .RoleName }}`
	p, err = r.ProfileName(settings)
	assert.NoError(t, err)
	assert.Equal(t, "OurCompany_Control_Tower_Playground:AWSAdministratorAccess", p)
}

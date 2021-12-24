package storage

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
	"io/ioutil"
	"os"
	"testing"

	// "github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const TEST_JSON_STORE_FILE = "./testdata/store.json"

type JsonStoreTestSuite struct {
	suite.Suite
	json     *JsonStore
	jsonFile string
}

func TestJsonStoreSuite(t *testing.T) {
	s := &JsonStoreTestSuite{}
	suite.Run(t, s)
}

func (s *JsonStoreTestSuite) SetupTest() {
	t := s.T()

	f, err := os.CreateTemp("", "*")
	assert.Nil(t, err)

	s.jsonFile = f.Name()
	f.Close()

	input, err := ioutil.ReadFile(TEST_JSON_STORE_FILE)
	assert.Nil(t, err)

	err = ioutil.WriteFile(s.jsonFile, input, 0600)
	assert.Nil(t, err)

	s.json, err = OpenJsonStore(s.jsonFile)
	assert.Nil(t, err)
}

func (s *JsonStoreTestSuite) AfterTest() {
	os.Remove(s.jsonFile)
}

func (s *JsonStoreTestSuite) TestRegisterClientData() {
	t := s.T()
	rcd := RegisterClientData{}

	err := s.json.GetRegisterClientData("foobar", &rcd)
	assert.NotNil(t, err)

	key := "us-east-1|https://d-xxxxxxx.awsapps.com/start"
	err = s.json.GetRegisterClientData(key, &rcd)
	assert.Nil(t, err)
	rcdTest := RegisterClientData{
		ClientId:              "not a real client id",
		ClientIdIssuedAt:      1629947379,
		ClientSecret:          "not a real secret",
		ClientSecretExpiresAt: 1637723379,
	}
	assert.Equal(t, rcdTest, rcd)

	err = s.json.SaveRegisterClientData(key, rcd)
	assert.Nil(t, err)
	assert.Equal(t, rcdTest, rcd)

	err = s.json.DeleteRegisterClientData(key)
	assert.Nil(t, err)

	err = s.json.GetRegisterClientData(key, &rcd)
	assert.NotNil(t, err)
}

func (s *JsonStoreTestSuite) TestRoleCredentials() {
	t := s.T()
	rc := RoleCredentials{}
	arn := "arn:aws:iam::012344553243:role/AWSAdministratorAccess"

	err := s.json.GetRoleCredentials("foobar", &rc)
	assert.NotNil(t, err)

	err = s.json.GetRoleCredentials(arn, &rc)
	assert.Nil(t, err)

	rcTest := RoleCredentials{
		RoleName:        "AWSAdministratorAccess",
		AccountId:       12344553243,
		AccessKeyId:     "not a real access key id",
		SecretAccessKey: "not a real acess key",
		SessionToken:    "not a real session token",
		Expiration:      1637444478000,
	}
	assert.Equal(t, rcTest, rc)

	err = s.json.SaveRoleCredentials(arn, rc)
	assert.Nil(t, err)
	assert.Equal(t, rcTest, rc)

	err = s.json.DeleteRoleCredentials(arn)
	assert.Nil(t, err)

	err = s.json.GetRoleCredentials(arn, &rc)
	assert.NotNil(t, err)
}

func (s *JsonStoreTestSuite) TestCreateTokenResponse() {
	t := s.T()
	tr := CreateTokenResponse{}
	key := "us-east-1|https://d-xxxxxxx.awsapps.com/start"

	err := s.json.GetCreateTokenResponse("foobar", &tr)
	assert.NotNil(t, err)

	err = s.json.GetCreateTokenResponse(key, &tr)
	assert.Nil(t, err)

	trTest := CreateTokenResponse{
		AccessToken:  "not a real token",
		ExpiresIn:    28800,
		ExpiresAt:    1637469677,
		IdToken:      "",
		RefreshToken: "",
		TokenType:    "Bearer",
	}
	assert.Equal(t, trTest, tr)

	err = s.json.SaveCreateTokenResponse(key, tr)
	assert.Nil(t, err)
	assert.Equal(t, trTest, tr)

	err = s.json.DeleteCreateTokenResponse(key)
	assert.Nil(t, err)

	err = s.json.GetCreateTokenResponse(key, &tr)
	assert.NotNil(t, err)
}

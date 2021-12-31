package storage

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

	// "github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/aws-sso-cli/utils"
)

// JsonStore implements SecureStorage insecurely
type JsonStore struct {
	filename            string
	RegisterClient      map[string]RegisterClientData  `json:"RegisterClient,omitempty"`
	StartDeviceAuth     map[string]StartDeviceAuthData `json:"StartDeviceAuth,omitempty"`
	CreateTokenResponse map[string]CreateTokenResponse `json:"CreateTokenResponse,omitempty"`
	RoleCredentials     map[string]RoleCredentials     `json:"RoleCredentials,omitempty"`
}

// OpenJsonStore opens our insecure JSON storage backend
func OpenJsonStore(fileName string) (*JsonStore, error) {
	cache := JsonStore{
		filename:            fileName,
		RegisterClient:      map[string]RegisterClientData{},
		StartDeviceAuth:     map[string]StartDeviceAuthData{},
		CreateTokenResponse: map[string]CreateTokenResponse{},
		RoleCredentials:     map[string]RoleCredentials{},
	}

	cacheBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Infof("Creating new cache file: %s", fileName)
	} else {
		err = json.Unmarshal(cacheBytes, &cache)
	}

	return &cache, err
}

// save writes the JSON store file, creating the directory if necessary
func (jc *JsonStore) save() error {
	log.Debugf("Saving JSON Cache")
	jbytes, err := json.MarshalIndent(jc, "", "  ")
	if err != nil {
		log.WithError(err).Errorf("Unable to marshal json")
		return err
	}
	err = utils.EnsureDirExists(jc.filename)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(jc.filename, jbytes, 0600)
}

// SaveRegisterClientData saves the RegisterClientData in our JSON store
func (jc *JsonStore) SaveRegisterClientData(key string, client RegisterClientData) error {
	jc.RegisterClient[key] = client
	return jc.save()
}

// GetRegisterClientData retrieves the RegisterClientData from our JSON store
func (jc *JsonStore) GetRegisterClientData(key string, client *RegisterClientData) error {
	var ok bool
	*client, ok = jc.RegisterClient[key]
	if !ok {
		return fmt.Errorf("No RegisterClientData for %s", key)
	}
	return nil
}

// DeleteRegisterClientData deletes the RegisterClientData from the JSON store
func (jc *JsonStore) DeleteRegisterClientData(key string) error {
	delete(jc.RegisterClient, key)
	return jc.save()
}

// SaveCreateTokenResponse stores the token in the json file
func (jc *JsonStore) SaveCreateTokenResponse(key string, token CreateTokenResponse) error {
	jc.CreateTokenResponse[key] = token
	return jc.save()
}

// GetCreateTokenResponse retrieves the CreateTokenResponse from the json file
func (jc *JsonStore) GetCreateTokenResponse(key string, token *CreateTokenResponse) error {
	var ok bool
	*token, ok = jc.CreateTokenResponse[key]
	if !ok {
		return fmt.Errorf("No CreateTokenResponse for %s", key)
	}
	return nil
}

// DeleteCreateTokenResponse deletes the token from the json file
func (jc *JsonStore) DeleteCreateTokenResponse(key string) error {
	delete(jc.CreateTokenResponse, key)
	return jc.save()
}

// SaveRoleCredentials stores the token in the json file
func (jc *JsonStore) SaveRoleCredentials(arn string, token RoleCredentials) error {
	jc.RoleCredentials[arn] = token
	return jc.save()
}

// GetRoleCredentials retrieves the RoleCredentials from the json file
func (jc *JsonStore) GetRoleCredentials(arn string, token *RoleCredentials) error {
	var ok bool
	*token, ok = jc.RoleCredentials[arn]
	if !ok {
		return fmt.Errorf("No RoleCredentials for %s", arn)
	}
	return nil
}

// DeleteRoleCredentials deletes the token from the json file
func (jc *JsonStore) DeleteRoleCredentials(arn string) error {
	delete(jc.RoleCredentials, arn)
	return jc.save()
}

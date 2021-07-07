package main

/*
 * AWS SSO CLI
 * Copyright (c) 2021 Aaron Turner  <aturner at synfin dot net>
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
	"fmt"
	"io/ioutil"
	"reflect"
	"sort"

	"github.com/Songmu/prompter"
	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/onelogin-aws-role/utils"
	yaml "gopkg.in/yaml.v3"
)

// Fields match those in FlatConfig.  Used when user doesn't have the `fields` in
// their YAML config file or provided list on the CLI
var defaultListFields = []string{
	"AccountId",
	"AccountName",
	"RoleName",
}

var allListFields = map[string]string{
	"Id":           "Column Index",
	"AccountId":    "AWS AccountID",
	"AccountName":  "AWS AccountName",
	"EmailAddress": "Root Email",
	"RoleName":     "AWS Role",
	"Expires":      "Creds Expire",
	"Profile":      "AWS_PROFILE",
}

type ListCmd struct {
	Fields      []string `kong:"optional,arg,enum='Id,AccountId,AccountName,EmailAddress,RoleName,Expires,Profile',help='Fields to display',env='AWS_SSO_FIELDS'"`
	ListFields  bool     `kong:"optional,name='list-fields',short='f',help='List available fields'"`
	ForceUpdate bool     `kong:"optional,name='force-update',help='Force account/role cache update'"`
}

// what should this actually do?
func (cc *ListCmd) Run(ctx *RunContext) error {
	var err error

	// If `-f` then print our fields and exit
	if ctx.Cli.List.ListFields {
		listAllFields()
		return nil
	}

	roles := map[string][]RoleInfo{}
	err = ctx.Store.GetRoles(&roles)

	if err != nil || ctx.Cli.List.ForceUpdate {
		roles = map[string][]RoleInfo{} // zero out roles if we are doing a --force-update
		sso := ctx.Config.SSO[ctx.Cli.SSO]
		awssso := NewAWSSSO(sso.SSORegion, sso.StartUrl, &ctx.Store)
		err = awssso.Authenticate(ctx.Cli.PrintUrl, ctx.Cli.Browser)
		if err != nil {
			log.WithError(err).Fatalf("Unable to authenticate")
		}

		accounts, err := awssso.GetAccounts()
		if err != nil {
			log.WithError(err).Fatalf("Unable to get accounts")
		}

		for _, a := range accounts {
			account := a.AccountId
			roleInfo, err := awssso.GetRoles(a)
			if err != nil {
				log.WithError(err).Fatalf("Unable to get roles for AccountId: %s", account)
			}

			for _, r := range roleInfo {
				roles[account] = append(roles[account], r)
			}
		}
		ctx.Store.SaveRoles(roles)

		// now update our config.yaml
		changes, err := sso.UpdateRoles(roles)
		if err != nil {
			log.WithError(err).Fatalf("Unable to update our config file")
		}
		if changes > 0 {
			p := fmt.Sprintf("Update config file with %d new roles?", changes)
			if prompter.YN(p, true) {
				b, _ := yaml.Marshal(ctx.Config)
				cfile := fmt.Sprintf("%s", ctx.Cli.ConfigFile)
				err = ioutil.WriteFile(cfile, b, 0644)
				if err != nil {
					log.WithError(err).Fatalf("Unable to save config %s", cfile)
				}
			}
		}

	} else {
		log.Info("Using cache.  Use --force-update to force a cache update.")
	}

	fields := defaultListFields
	if len(ctx.Cli.List.Fields) > 0 {
		fields = ctx.Cli.List.Fields
	}

	printRoles(roles, fields)

	return nil
}

// Print all our roles
func printRoles(roles map[string][]RoleInfo, fields []string) []RoleInfo {
	ret := []RoleInfo{}
	tr := []utils.TableStruct{}
	idx := 0

	// print in AccountId order
	accounts := []string{}
	for account, _ := range roles {
		accounts = append(accounts, account)
	}
	sort.Strings(accounts)

	for _, account := range accounts {
		for _, role := range roles[account] {
			role.Id = idx
			idx += 1
			tr = append(tr, role)
			ret = append(ret, role)
		}
	}

	utils.GenerateTable(tr, fields)
	fmt.Printf("\n")
	return ret
}

// Code to --list-fields
type ConfigFieldNames struct {
	Field       string `header:"Field"`
	Description string `header:"Description"`
}

func (cfn ConfigFieldNames) GetHeader(fieldName string) (string, error) {
	v := reflect.ValueOf(cfn)
	return utils.GetHeaderTag(v, fieldName)
}

func listAllFields() {
	ts := []utils.TableStruct{}
	for k, v := range allListFields {
		ts = append(ts, ConfigFieldNames{
			Field:       k,
			Description: v,
		})
	}

	fields := []string{"Field", "Description"}
	utils.GenerateTable(ts, fields)
	fmt.Printf("\n")
}

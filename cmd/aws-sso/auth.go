package main

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
	"fmt"

	"github.com/synfinatic/aws-sso-cli/internal/url"
	"github.com/synfinatic/aws-sso-cli/internal/utils"
	"github.com/synfinatic/aws-sso-cli/sso"
)

type SelectCliArgs struct {
	Arn       string
	AccountId int64
	RoleName  string
	Profile   string
}

func NewSelectCliArgs(arn string, accountId int64, role, profile string) *SelectCliArgs {
	return &SelectCliArgs{
		Arn:       arn,
		AccountId: accountId,
		RoleName:  role,
		Profile:   profile,
	}
}

func (a *SelectCliArgs) Update(ctx *RunContext) (*sso.AWSSSO, error) {
	if a.AccountId != 0 && a.RoleName != "" {
		return doAuth(ctx), nil
	} else if a.Profile != "" {
		awssso := doAuth(ctx)
		cache := ctx.Settings.Cache.GetSSO()
		rFlat, err := cache.Roles.GetRoleByProfile(a.Profile, ctx.Settings)
		if err != nil {
			return awssso, err
		}

		a.AccountId = rFlat.AccountId
		a.RoleName = rFlat.RoleName

		return awssso, nil
	} else if a.Arn != "" {
		awssso := doAuth(ctx)
		accountId, role, err := utils.ParseRoleARN(a.Arn)
		if err != nil {
			return awssso, err
		}
		a.AccountId = accountId
		a.RoleName = role

		return awssso, nil
	}
	return &sso.AWSSSO{}, fmt.Errorf("Please specify both --account and --role")
}

// Creates a singleton AWSSO object post authentication
func doAuth(ctx *RunContext) *sso.AWSSSO {
	if AwsSSO != nil {
		return AwsSSO
	}
	s, err := ctx.Settings.GetSelectedSSO(ctx.Cli.SSO)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	AwsSSO = sso.NewAWSSSO(s, &ctx.Store)
	err = AwsSSO.Authenticate(ctx.Settings.UrlAction, ctx.Settings.Browser)
	if err != nil {
		log.WithError(err).Fatalf("Unable to authenticate")
	}
	if err = ctx.Settings.Cache.Expired(s); err != nil {
		ssoName, err := ctx.Settings.GetSelectedSSOName(ctx.Cli.SSO)
		log.Infof("Refreshing AWS SSO role cache for %s, please wait...", ssoName)
		if err != nil {
			log.Fatalf(err.Error())
		}
		if err = ctx.Settings.Cache.Refresh(AwsSSO, s, ssoName); err != nil {
			log.WithError(err).Fatalf("Unable to refresh cache")
		}
		if err = ctx.Settings.Cache.Save(true); err != nil {
			log.WithError(err).Errorf("Unable to save cache")
		}

		// should we update our config??
		if !ctx.Cli.NoConfigCheck && ctx.Settings.AutoConfigCheck {
			if ctx.Settings.ConfigProfilesUrlAction != url.ConfigProfilesUndef {
				cfgFile := utils.GetHomePath("~/.aws/config")

				action, _ := url.NewAction(string(ctx.Settings.ConfigProfilesUrlAction))
				profiles, err := ctx.Settings.GetAllProfiles(action)
				if err != nil {
					log.Warnf("Unable to update %s: %s", cfgFile, err.Error())
					return AwsSSO
				}

				if err = profiles.UniqueCheck(ctx.Settings); err != nil {
					log.Errorf("Unable to update %s: %s", cfgFile, err.Error())
					return AwsSSO
				}

				f, err := utils.NewFileEdit(CONFIG_TEMPLATE, profiles)
				if err != nil {
					log.Errorf("%s", err)
					return AwsSSO
				}

				if err = f.UpdateConfig(true, false, cfgFile); err != nil {
					log.Errorf("Unable to update %s: %s", cfgFile, err.Error())
					return AwsSSO
				}
			}
		}
	}
	return AwsSSO
}

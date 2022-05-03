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
	"os"
	"runtime"
	"strings"

	// log "github.com/sirupsen/logrus"
	"github.com/synfinatic/aws-sso-cli/internal/utils"
)

type EvalCmd struct {
	// AWS Params
	Arn       string `kong:"short='a',help='ARN of role to assume',predictor='arn'"`
	AccountId int64  `kong:"name='account',short='A',help='AWS AccountID of role to assume',predictor='accountId'"`
	Role      string `kong:"short='R',help='Name of AWS Role to assume',predictor='role'"`
	Profile   string `kong:"short='p',help='Name of AWS Profile to assume',predictor='profile'"`

	Clear    bool   `kong:"short='c',help='Generate \"unset XXXX\" commands to clear environment'"`
	NoRegion bool   `kong:"short='n',help='Do not set/clear AWS_DEFAULT_REGION from config.yaml'"`
	Refresh  bool   `kong:"short='r',help='Refresh current IAM credentials'"`
	EnvArn   string `kong:"hidden,env='AWS_SSO_ROLE_ARN'"` // used for refresh
}

func (cc *EvalCmd) Run(ctx *RunContext) error {
	var err error

	if runtime.GOOS == "windows" && !strings.HasSuffix(os.Getenv("SHELL"), "/bash") {
		return fmt.Errorf("eval is not supported on Windows unless running under bash")
	}

	var role string
	var accountid int64

	if ctx.Cli.Eval.Clear {
		return unsetEnvVars(ctx)
	}

	// refreshing?
	if ctx.Cli.Eval.Refresh {
		if ctx.Cli.Eval.EnvArn != "" {
			return fmt.Errorf("Unable to determine current IAM role")
		}
		accountid, role, err = utils.ParseRoleARN(ctx.Cli.Eval.EnvArn)
		if err != nil {
			return err
		}
	} else if ctx.Cli.Eval.Profile != "" {
		cache := ctx.Settings.Cache.GetSSO()
		rFlat, err := cache.Roles.GetRoleByProfile(ctx.Cli.Eval.Profile, ctx.Settings)
		if err != nil {
			return err
		}

		role = rFlat.RoleName
		accountid = rFlat.AccountId
	} else if ctx.Cli.Eval.Arn != "" {
		accountid, role, err = utils.ParseRoleARN(ctx.Cli.Eval.Arn)
		if err != nil {
			return err
		}
	} else if ctx.Cli.Eval.Role != "" && ctx.Cli.Eval.AccountId > 0 {
		// if CLI args are speecified, use that
		role = ctx.Cli.Eval.Role
		accountid = ctx.Cli.Eval.AccountId
	} else {
		return fmt.Errorf("Please specify --refresh, --clear, --arn, or --account and --role")
	}
	region := ctx.Settings.GetDefaultRegion(accountid, role, ctx.Cli.Eval.NoRegion)

	awssso := doAuth(ctx)

	for k, v := range execShellEnvs(ctx, awssso, accountid, role, region) {
		if len(v) == 0 {
			fmt.Printf("unset %s\n", k)
		} else {
			fmt.Printf("export %s=\"%s\"\n", k, v)
		}
	}
	return nil
}

func unsetEnvVars(ctx *RunContext) error {
	envs := []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_SSO_ACCOUNT_ID",
		"AWS_SSO_ROLE_NAME",
		"AWS_SSO_ROLE_ARN",
		"AWS_SSO_SESSION_EXPIRATION",
		"AWS_SSO_PROFILE",
		"AWS_SSO",
	}

	// clear the region if
	// 1. User did not specify --no-region AND
	// 2. The AWS_DEFAULT_REGION is managed by us (tracks AWS_SSO_DEFAULT_REGION)
	if !ctx.Cli.Eval.NoRegion && os.Getenv("AWS_DEFAULT_REGION") == os.Getenv("AWS_SSO_DEFAULT_REGION") {
		envs = append(envs, "AWS_DEFAULT_REGION")
		envs = append(envs, "AWS_SSO_DEFAULT_REGION")
	} else if os.Getenv("AWS_DEFAULT_REGION") != os.Getenv("AWS_SSO_DEFAULT_REGION") {
		// clear the tracking variable if we don't match
		envs = append(envs, "AWS_SSO_DEFAULT_REGION")
	}

	for _, env := range ctx.Settings.GetEnvVarTags() {
		envs = append(envs, env)
	}

	for _, e := range envs {
		fmt.Printf("unset %s\n", e)
	}
	return nil
}

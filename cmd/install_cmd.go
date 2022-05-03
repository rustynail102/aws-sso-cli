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

	"github.com/synfinatic/aws-sso-cli/internal/helper"
	"github.com/willabides/kongplete"
)

type InstallCmd struct {
}

func (cc *InstallCmd) Run(ctx *RunContext) error {
	kp := &kongplete.InstallCompletions{}
	err := kp.Run(ctx.Kctx)

	// don't exit
	if err != nil {
		fmt.Printf(err.Error())
	}

	return helper.WriteHelper()
}

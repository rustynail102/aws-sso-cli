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
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	//	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/aws-sso-cli/sso"
)

type CompleterExec = func(*RunContext, *sso.AWSSSO, int64, string) error

type TagsCompleter struct {
	ctx      *RunContext
	sso      *sso.SSOConfig
	roleTags *sso.RoleTags
	allTags  *sso.TagsList
	suggest  []prompt.Suggest
	exec     CompleterExec
}

func NewTagsCompleter(ctx *RunContext, s *sso.SSOConfig, exec CompleterExec) *TagsCompleter {
	roleTags := ctx.Cache.Roles.GetRoleTagsSelect()
	allTags := ctx.Cache.Roles.GetAllTagsSelect()

	return &TagsCompleter{
		ctx:      ctx,
		sso:      s,
		roleTags: roleTags,
		allTags:  allTags,
		suggest:  completeTags(roleTags, allTags, []string{}),
		exec:     exec,
	}
}

func (tc *TagsCompleter) Complete(d prompt.Document) []prompt.Suggest {
	if d.TextBeforeCursor() == "" {
		return prompt.FilterHasPrefix(tc.suggest, d.GetWordBeforeCursor(), true)
	}

	args := d.TextBeforeCursor()
	w := d.GetWordBeforeCursor()

	argsList := strings.Split(args, " ")
	suggest := completeTags(tc.roleTags, tc.allTags, argsList)
	return prompt.FilterHasPrefix(suggest, w, true)
	// return prompt.FilterFuzzy(suggest, w, true)
}

func (tc *TagsCompleter) Executor(args string) {
	if args == "exit" {
		os.Exit(1)
	}
	argsMap, _, _ := argsToMap(strings.Split(args, " "))

	ssoRoles := tc.roleTags.GetMatchingRoles(argsMap)
	if len(ssoRoles) == 0 {
		log.Fatalf("No matching roles")
	} else if len(ssoRoles) > 1 {
		log.Fatalf("Invalid selection")
	}

	aId, rName, err := sso.GetRoleParts(ssoRoles[0])
	if err != nil {
		log.Fatalf("Unable to parse %s: %s", ssoRoles[0], err.Error())
	}
	awsSSO := doAuth(tc.ctx)
	err = tc.exec(tc.ctx, awsSSO, aId, rName)
	if err != nil {
		log.Fatalf("Unable to exec: %s", err.Error())
	}
	return
}

// completeExitChecker impliments prompt.ExitChecker
func (tc *TagsCompleter) ExitChecker(in string, breakline bool) bool {
	return breakline // exit our Run() loop after user selects something
}

// return a list of suggestions based on user selected []key:value
func completeTags(roleTags *sso.RoleTags, allTags *sso.TagsList, args []string) []prompt.Suggest {
	suggestions := []prompt.Suggest{}

	currentTags, nextKey, nextValue := argsToMap(args)
	if roleTags.GetMatchCount(currentTags) == 1 {
		return suggestions // empty list if we have a single role
	}

	if nextKey == "" {
		// Find roles which match selection & remaining Tag keys
		currentRoles := roleTags.GetMatchingRoles(currentTags)

		selectedKeys := []string{}
		for k, _ := range currentTags {
			selectedKeys = append(selectedKeys, k)
		}

		returnedRoles := map[string]bool{}

		for _, key := range allTags.UniqueKeys(selectedKeys) {
			uniqueRoles := roleTags.GetPossibleUniqueRoles(currentTags, key, (*allTags)[key])
			if len(args) > 0 && len(uniqueRoles) == len(currentRoles) {
				// skip keys which can't reduce our options
				for _, role := range uniqueRoles {
					if _, ok := returnedRoles[role]; ok {
						// don't return the same role multiple times
						continue
					}
					suggestions = append(suggestions, prompt.Suggest{
						Text:        role,
						Description: "",
					})
					returnedRoles[role] = true
				}
				continue
			}
			suggestions = append(suggestions, prompt.Suggest{
				Text: key,
				Description: fmt.Sprintf("%d roles/%d choices", len(uniqueRoles),
					len(allTags.UniqueValues(key))),
			})
		}
	} else if nextValue == "" {
		// We have a 'nextKey', so search for Tags which match
		values := (*allTags).UniqueValues(nextKey)
		if len(values) > 0 {
			// found exact match for our nextKey
			for _, value := range values {
				checkArgs := []string{}
				for _, v := range args {
					if v != "" { // don't include the empty
						checkArgs = append(checkArgs, v)
					}
				}
				checkArgs = append(checkArgs, value)
				checkArgs = append(checkArgs, "") // mark value as "complete"
				argsMap, _, _ := argsToMap(checkArgs)
				checkRoles := roleTags.GetMatchingRoles(argsMap)
				roleCnt := len(checkRoles)
				desc := ""
				switch roleCnt {
				case 0:
					continue

				case 1:
					desc = checkRoles[0]

				default:
					desc = fmt.Sprintf("%d roles", roleCnt)

				}
				suggestions = append(suggestions, prompt.Suggest{
					Text:        value,
					Description: desc,
				})
			}
		} else {
			// no exact match, look for the key

			usedKeys := []string{}
			for k, _ := range currentTags {
				usedKeys = append(usedKeys, k)
			}
			remainKeys := allTags.UniqueKeys(usedKeys)

			for _, checkKey := range remainKeys {
				if strings.Contains(strings.ToLower(checkKey), strings.ToLower(nextKey)) {
					suggestions = append(suggestions, prompt.Suggest{
						Text:        checkKey,
						Description: fmt.Sprintf("%d choices", len(allTags.UniqueValues(checkKey))),
					})
				}
			}
		}
	} else {
		// We have a 'nextValue', so search for Tags which match
		for _, checkValue := range allTags.UniqueValues(nextKey) {
			if strings.Contains(strings.ToLower(checkValue), strings.ToLower(nextValue)) {
				testSet := map[string]string{}
				for k, v := range currentTags {
					testSet[k] = v
				}
				testSet[nextKey] = checkValue
				matchedRoles := roleTags.GetMatchingRoles(testSet)
				matchedCnt := len(matchedRoles)
				if matchedCnt > 0 {
					suggestions = append(suggestions, prompt.Suggest{
						Text:        checkValue,
						Description: fmt.Sprintf("%d roles", matchedCnt),
					})
				}
			}
		}
	}
	return suggestions
}

// Converts a list of 'key value' strings to a key/value map and uncompleted key/value pair
func argsToMap(args []string) (map[string]string, string, string) {
	tags := map[string]string{}
	retKey := ""
	retValue := ""
	cleanArgs := []string{}
	completeWord := false

	// remove any empty strings
	for _, a := range args {
		if a != "" {
			cleanArgs = append(cleanArgs, a)
		}
	}

	if len(cleanArgs) == 0 {
		return map[string]string{}, "", ""
	} else if len(cleanArgs) == 1 {
		return map[string]string{}, cleanArgs[0], ""
	}

	// our last word is complete
	if args[len(args)-1] == "" {
		completeWord = true
	}

	if len(cleanArgs)%2 == 0 && completeWord {
		// we have a complete set of key => value pairs
		for i := 0; i < len(args)-1; i += 2 {
			tags[cleanArgs[i]] = cleanArgs[i+1]
		}
	} else if len(cleanArgs)%2 == 0 {
		// final word is an incomplete value
		for i := 0; i < len(cleanArgs)-2; i += 2 {
			tags[cleanArgs[i]] = cleanArgs[i+1]
		}
		retKey = cleanArgs[len(cleanArgs)-2]
		retValue = cleanArgs[len(cleanArgs)-1]
	} else {
		// final word is a (part of a) key
		retKey = cleanArgs[len(cleanArgs)-1]
		cleanArgs = cleanArgs[:len(cleanArgs)-1]
		for i := 0; i < len(cleanArgs)-2; i += 2 {
			tags[cleanArgs[i]] = cleanArgs[i+1]
		}
	}
	return tags, retKey, retValue
}

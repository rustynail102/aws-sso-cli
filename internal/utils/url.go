package utils

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
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/skratchdot/open-golang/open" // default opener
)

// taken from https://github.com/honsiorovskyi/open-url-in-container/blob/1.0.3/launcher.sh
var FIREFOX_PLUGIN_COLORS []string = []string{
	"blue",
	"turquoise",
	"green",
	"yellow",
	"orange",
	"red",
	"pink",
	"purple",
	// "toolbar",  not a valid input, even if it is user selectable
}

var FIREFOX_PLUGIN_ICONS []string = []string{
	"fingerprint",
	"briefcase",
	"dollar",
	"cart",
	"gift",
	"vacation",
	"food",
	"fruit",
	"pet",
	"tree",
	"chill",
	"circle",
	// "fence",  not a valid input, even if it is user selectable
}

const FIREFOX_CONTAINER_FORMAT = "ext+container:name=%s&url=%s&color=%s&icon=%s"

// FirefoxContainerUrl generates a URL for Firefox Containers
func FirefoxContainerUrl(target, name, color, icon string) string {
	if !StrListContains(color, FIREFOX_PLUGIN_COLORS) {
		log.Warnf("Invalid Firefox Container color: %s", color)
		color = FIREFOX_PLUGIN_COLORS[0]
	}

	if !StrListContains(icon, FIREFOX_PLUGIN_ICONS) {
		log.Warnf("Invalid Firefox Container icon: %s", icon)
		icon = FIREFOX_PLUGIN_ICONS[0]
	}

	return fmt.Sprintf(FIREFOX_CONTAINER_FORMAT, name, url.QueryEscape(target), color, icon)
}

var printWriter io.Writer = os.Stderr

func execWithUrl(command interface{}, url string) error {
	var cmd *exec.Cmd
	var cmdStr string

	switch command.(type) {
	case []interface{}:
		program := ""
		x := []string{}
		for _, iface := range command.([]interface{}) {
			v := iface.(string)
			if strings.Contains(v, "%s") {
				if program == "" {
					program = fmt.Sprintf(v, url)
				} else {
					x = append(x, fmt.Sprintf(v, url))
				}
			} else {
				if program == "" {
					program = v
				} else {
					x = append(x, v)
				}
			}
		}
		cmdStr = fmt.Sprintf("%s %s", program, strings.Join(x, " "))
		log.Debugf("exec command as array: %s", cmdStr)
		cmd = exec.Command(program, x...)
	default:
		return fmt.Errorf("Invalid UrlExecCommand type: %v", command)
	}

	//	var stderr bytes.Buffer
	//	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		err = fmt.Errorf("Unable to exec `%s`: %s", cmdStr, err)
	}
	log.Debugf("Opened our URL with %s", command.([]interface{})[0])
	return err
}

// these types & variables make our code easier to unit test
type urlOpenerFunc func(string) error
type urlOpenerWithFunc func(string, string) error
type clipboardWriterFunc func(string) error

var urlOpener urlOpenerFunc = open.Run
var urlOpenerWith urlOpenerWithFunc = open.RunWith
var clipboardWriter clipboardWriterFunc = clipboard.WriteAll

type UrlAction int

const (
	UrlActionClip     UrlAction = iota // copy to clipboard
	UrlActionPrint                     // print message & url to stderr
	UrlActionPrintUrl                  // print only the  url to stderr
	UrlActionExec                      // Exec comand
	UrlActionOpen                      // auto-open in default or specified browser
)

type HandleUrl struct {
	Action  UrlAction
	ExecCmd interface{}
	Browser string
	Url     string
	PreMsg  string
	PostMsg string
}

func NewHandleUrl(action, browser string, command interface{}) *HandleUrl {
	var a UrlAction

	switch action {
	case "clip":
		a = UrlActionClip
	case "print":
		a = UrlActionPrint
	case "printurl":
		a = UrlActionPrintUrl
	case "exec":
		a = UrlActionExec
	case "open":
		a = UrlActionOpen
	default:
		log.Panicf("invalid --url-action: %s", action)
	}

	h := &HandleUrl{
		Action:  a,
		Browser: browser,
		ExecCmd: command,
	}
	return h
}

// Prints, opens or copies to clipboard the given URL
func (h *HandleUrl) Open(url, pre, post string) error {
	var err error
	switch h.Action {
	case UrlActionClip:
		err = clipboardWriter(url)
		if err == nil {
			log.Infof("Please open URL copied to clipboard.\n")
		} else {
			err = fmt.Errorf("Unable to copy URL to clipboard: %s", err.Error())
		}
	case UrlActionExec:
		err = execWithUrl(h.ExecCmd, url)
	case UrlActionPrint:
		fmt.Fprintf(printWriter, "%s%s%s", pre, url, post)
	case UrlActionPrintUrl:
		fmt.Fprintf(printWriter, "%s\n", url)
	case UrlActionOpen:
		var browser string
		switch h.Browser {
		case "":
			err = urlOpener(url)
			browser = "default browser"
		default:
			err = urlOpenerWith(url, h.Browser)
		}
		if err != nil {
			err = fmt.Errorf("Unable to open URL with %s: %s", browser, err.Error())
		} else {
			log.Infof("Opening URL in %s.\n", browser)
		}
	}

	return err
}

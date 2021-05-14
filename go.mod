module github.com/synfinatic/aws-sso-cli

go 1.16

// pin this version (or later) until 99designs/keyring updates.
replace github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4

require (
	github.com/99designs/keyring v1.1.6
	github.com/Songmu/prompter v0.4.0
	github.com/alecthomas/kong v0.2.15
	github.com/aws/aws-sdk-go v1.36.23
	github.com/sirupsen/logrus v1.7.0
)

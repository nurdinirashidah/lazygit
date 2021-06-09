package commands

import (
	"regexp"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	. "github.com/jesseduffield/lazygit/pkg/commands/types"
)

func (c *GitCommand) SetCredentialHandlers(promptUserForCredential func(CredentialKind) string, handleCredentialError func(error)) {
	c.promptUserForCredential = promptUserForCredential
	c.handleCredentialError = handleCredentialError
}

// RunCommandWithCredentialsPrompt detect a username / password / passphrase question in a command
// promptUserForCredential is a function that gets executed when this function detect you need to fillin a password or passphrase
// The promptUserForCredential argument will be "username", "password" or "passphrase" and expects the user's password/passphrase or username back
func (c *GitCommand) RunCommandWithCredentialsPrompt(cmdObj *oscommands.CmdObj) error {
	ttyText := ""
	err := c.oSCommand.RunCommandAndParseOutput(cmdObj, func(word string) string {
		ttyText = ttyText + " " + word

		prompts := map[string]CredentialKind{
			`.+'s password:`:                         PASSWORD,
			`Password\s*for\s*'.+':`:                 PASSWORD,
			`Username\s*for\s*'.+':`:                 USERNAME,
			`Enter\s*passphrase\s*for\s*key\s*'.+':`: PASSPHRASE,
		}

		for pattern, askFor := range prompts {
			if match, _ := regexp.MatchString(pattern, ttyText); match {
				ttyText = ""
				return c.promptUserForCredential(askFor)
			}
		}

		return ""
	})

	return err
}

// this goes one step beyond RunCommandWithCredentialsPrompt and handles a credential error
func (c *GitCommand) RunCommandWithCredentialsHandling(cmdObj *oscommands.CmdObj) error {
	err := c.RunCommandWithCredentialsPrompt(cmdObj)
	c.handleCredentialError(err)
	return nil
}

func (c *GitCommand) FailOnCredentialsRequest(cmdObj *oscommands.CmdObj) *oscommands.CmdObj {
	lazyGitPath := c.GetOSCommand().GetLazygitPath()

	cmdObj.AddEnvVars(
		"LAZYGIT_CLIENT_COMMAND=EXIT_IMMEDIATELY",
		// prevents git from prompting us for input which would freeze the program. Only works for git v2.3+
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS="+lazyGitPath,
	)

	return cmdObj
}

type PushOpts struct {
	Force             bool
	SetUpstream       bool
	DestinationRemote string
	DestinationBranch string
}

func (c *GitCommand) Push(opts PushOpts) (bool, error) {
	cmdObj := BuildGitCmdObj("push", []string{opts.DestinationRemote, opts.DestinationBranch},
		map[string]bool{
			"--follow-tags":      c.GetConfigValue("push.followTags") != "false",
			"--force-with-lease": opts.Force,
			"--set-upstream":     opts.SetUpstream,
		})

	err := c.RunCommandWithCredentialsPrompt(cmdObj)

	if isRejectionErr(err) {
		return true, nil
	}

	c.handleCredentialError(err)

	return false, nil
}

func isRejectionErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Updates were rejected")
}

type FetchOptions struct {
	RemoteName string
	BranchName string
}

// Fetch fetch git repo
func (c *GitCommand) Fetch(opts FetchOptions) error {
	cmdObj := GetFetchCommandObj(opts)

	return c.RunCommandWithCredentialsHandling(cmdObj)
}

// FetchInBackground fails if credentials are requested
func (c *GitCommand) FetchInBackground(opts FetchOptions) error {
	cmdObj := GetFetchCommandObj(opts)

	cmdObj = c.FailOnCredentialsRequest(cmdObj)
	return c.oSCommand.RunExecutable(cmdObj)
}

func GetFetchCommandObj(opts FetchOptions) *oscommands.CmdObj {
	return BuildGitCmdObj("fetch", []string{opts.RemoteName, opts.BranchName}, nil)
}

func (c *GitCommand) FastForward(branchName string, remoteName string, remoteBranchName string) error {
	cmdObj := BuildGitCmdObj("fetch", []string{remoteName, remoteBranchName + ":" + branchName}, nil)
	return c.RunCommandWithCredentialsHandling(cmdObj)
}

func (c *GitCommand) FetchRemote(remoteName string) error {
	cmdObj := BuildGitCmdObj("fetch", []string{remoteName}, nil)
	return c.RunCommandWithCredentialsHandling(cmdObj)
}

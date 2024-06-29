package help

import (
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"github.com/remotemobprogramming/mob/v4/say"
)

func Help(configuration config.Configuration) {
	output := configuration.CliName + ` enables a smooth Git handover

Basic Commands:
  start              Start session from base branch in wip branch
  next               Handover changes in wip branch to next person
  done               Squash all changes in wip branch to index in base branch
  reset              Remove local and remote wip branch
  clean              Removes all orphan wip branches

Basic Commands with Options:
  start [<minutes>]                      Start <minutes> minutes timer
    [--include-uncommitted-changes|-i]   Move uncommitted changes to wip branch
    [--branch|-b <branch-postfix>]       Set wip branch to 'mob/<base-branch>` + configuration.WipBranchQualifierSeparator + `<branch-postfix>'
    [--create]                           Create the remote branch
    [--room <room-name>]                 Set room name for timer.mob.sh once
  next
    [--stay|-s]                          Stay on wip branch (default)
    [--return-to-base-branch|-r]         Return to base branch
    [--message|-m <commit-message>]      Override commit message
  done
    [--no-squash]                        Squash no commits from wip branch, only merge wip branch
    [--squash]                           Squash all commits from wip branch
    [--squash-wip]                       Squash wip commits from wip branch, maintaining manual commits
  reset
    [--branch|-b <branch-postfix>]       Set wip branch to 'mob/<base-branch>` + configuration.WipBranchQualifierSeparator + `<branch-postfix>'

Timer Commands:
  timer <minutes>           Start a <minutes> timer
    [--room <room-name>]    Set room name for timer.mob.sh once
  timer open                Opens the timer website
    [--room <room-name>]    Set room name for timer.mob.sh once
  start <minutes>           Start mob session in wip branch and a <minutes> timer
  break <minutes>           Start a <minutes> break timer
  goal                      Gives you the current goal of your timer.mob.sh room
    [<your-goal>]           Sets the goal of your timer.mob.sh room
    [--delete]              Deletes the goal of your timer.mob.sh room

Short Commands (Options and descriptions as above):
  s                  Alias for 'start'
  n                  Alias for 'next'
  d                  Alias for 'done'
  b                  Alias for 'branch'
  t                  Alias for 'timer'
  g                  Alias for 'goal'

Get more information:
  status             Show status of the current session
  fetch              Fetch remote state
  branch             Show remote wip branches
  config             Show all configuration options
  version            Show tool version
  help               Show help

Other
  moo                Moo!

Add '--debug' to any option to enable verbose logging.
Need more help? Join the community at slack.mob.sh
`
	say.Say(output)
}

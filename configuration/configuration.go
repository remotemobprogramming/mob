package configuration

type Configuration struct {
	CliName                        string // override with MOB_CLI_NAME
	RemoteName                     string // override with MOB_REMOTE_NAME
	WipCommitMessage               string // override with MOB_WIP_COMMIT_MESSAGE
	GitHooksEnabled                bool   // override with MOB_GIT_HOOKS_ENABLED
	RequireCommitMessage           bool   // override with MOB_REQUIRE_COMMIT_MESSAGE
	VoiceCommand                   string // override with MOB_VOICE_COMMAND
	VoiceMessage                   string // override with MOB_VOICE_MESSAGE
	NotifyCommand                  string // override with MOB_NOTIFY_COMMAND
	NotifyMessage                  string // override with MOB_NOTIFY_MESSAGE
	NextStay                       bool   // override with MOB_NEXT_STAY
	StartIncludeUncommittedChanges bool   // override with MOB_START_INCLUDE_UNCOMMITTED_CHANGES variable
	StartCreate                    bool   // override with MOB_START_CREATE variable
	StashName                      string // override with MOB_STASH_NAME
	WipBranchQualifier             string // override with MOB_WIP_BRANCH_QUALIFIER
	WipBranchQualifierSeparator    string // override with MOB_WIP_BRANCH_QUALIFIER_SEPARATOR
	WipBranchPrefix                string // override with MOB_WIP_BRANCH_PREFIX
	DoneSquash                     string // override with MOB_DONE_SQUASH
	OpenCommand                    string // override with MOB_OPEN_COMMAND
	Timer                          string // override with MOB_TIMER
	TimerRoom                      string // override with MOB_TIMER_ROOM
	TimerLocal                     bool   // override with MOB_TIMER_LOCAL
	TimerRoomUseWipBranchQualifier bool   // override with MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER
	TimerUser                      string // override with MOB_TIMER_USER
	TimerUrl                       string // override with MOB_TIMER_URL
	TimerInsecure                  bool   // override with MOB_TIMER_INSECURE
	ResetDeleteRemoteWipBranch     bool   // override with MOB_RESET_DELETE_REMOTE_WIP_BRANCH
}

func (c Configuration) Mob(command string) string {
	return c.CliName + " " + command
}

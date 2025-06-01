package terraform

type Command string

const (
	CommandFmt     Command = "fmt"
	CommandPlan    Command = "plan"
	CommandApply   Command = "apply"
	CommandDestroy Command = "destroy"
)

func (c Command) Validate() bool {
	switch c {
	case CommandFmt, CommandPlan, CommandApply, CommandDestroy:
		return true
	default:
		return false
	}
}

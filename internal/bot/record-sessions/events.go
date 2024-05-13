package recordsessions

import (
	"github.com/kvizyx/cycle"
)

const (
	EventTypeMemberJoin cycle.EventType = iota
	EventTypeMemberLeave
)

type EventMemberJoin struct{}

func (e EventMemberJoin) Type() cycle.EventType {
	return EventTypeMemberJoin
}

type EventMemberLeave struct{}

func (e EventMemberLeave) Type() cycle.EventType {
	return EventTypeMemberLeave
}

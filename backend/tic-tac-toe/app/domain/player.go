package domain

import googleuuid "github.com/google/uuid"

type PlayerKind string

const (
	PlayerKindUnknown  PlayerKind = ""
	PlayerKindUser     PlayerKind = "user"
	PlayerKindComputer PlayerKind = "computer"
)

const ComputerPlayerUUID = "computer"

type PlayerRef struct {
	Kind PlayerKind
	UUID googleuuid.UUID
}

func NewUserPlayerRef(uuid googleuuid.UUID) PlayerRef {
	return PlayerRef{Kind: PlayerKindUser, UUID: uuid}
}

func NewComputerPlayerRef() PlayerRef {
	return PlayerRef{Kind: PlayerKindComputer}
}

func PlayerRefFromString(value string) PlayerRef {
	switch value {
	case "":
		return PlayerRef{}
	case ComputerPlayerUUID:
		return NewComputerPlayerRef()
	default:
		uuid, err := googleuuid.Parse(value)
		if err != nil || uuid == googleuuid.Nil {
			return PlayerRef{}
		}
		return NewUserPlayerRef(uuid)
	}
}

func (p PlayerRef) String() string {
	switch p.Kind {
	case PlayerKindComputer:
		return ComputerPlayerUUID
	case PlayerKindUser:
		if p.UUID != googleuuid.Nil {
			return p.UUID.String()
		}
	}
	return ""
}

func (p PlayerRef) IsZero() bool {
	return p.Kind == PlayerKindUnknown && p.UUID == googleuuid.Nil
}

func (p PlayerRef) IsComputer() bool {
	return p.Kind == PlayerKindComputer
}

func (p PlayerRef) IsUser() bool {
	return p.Kind == PlayerKindUser && p.UUID != googleuuid.Nil
}

func (p PlayerRef) Matches(uuid googleuuid.UUID) bool {
	return p.IsUser() && p.UUID == uuid
}

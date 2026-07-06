package user

import (
	"encoding/json"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
)

const AggregateType = domain.AggregateUser

const (
	EventCreated         = "UserCreated"
	EventUpdated         = "UserUpdated"
	EventDeleted         = "UserDeleted"
	EventPasswordChanged = "PasswordChanged"
	EventEmailChanged    = "EmailChanged"
	EventAddressChanged  = "AddressChanged"
	EventAvatarChanged   = "AvatarChanged"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Address   string    `json:"address,omitempty"`
	Avatar    string    `json:"avatar,omitempty"`
	Deleted   bool      `json:"deleted,omitempty"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreatedPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdatedPayload struct {
	Name    *string `json:"name,omitempty"`
	Email   *string `json:"email,omitempty"`
	Address *string `json:"address,omitempty"`
	Avatar  *string `json:"avatar,omitempty"`
}

type EmailChangedPayload struct {
	Email string `json:"email"`
}

type AddressChangedPayload struct {
	Address string `json:"address"`
}

type AvatarChangedPayload struct {
	Avatar string `json:"avatar"`
}

func New(id string) *User {
	return &User{ID: id}
}

func (u *User) Apply(event domain.StoredEvent) error {
	u.Version = event.Version
	u.UpdatedAt = event.CreatedAt

	switch event.EventType {
	case EventCreated:
		var p CreatedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		u.Name = p.Name
		u.Email = p.Email
		u.CreatedAt = event.CreatedAt
		u.Deleted = false
	case EventUpdated:
		var p UpdatedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		if p.Name != nil {
			u.Name = *p.Name
		}
		if p.Email != nil {
			u.Email = *p.Email
		}
		if p.Address != nil {
			u.Address = *p.Address
		}
		if p.Avatar != nil {
			u.Avatar = *p.Avatar
		}
	case EventEmailChanged:
		var p EmailChangedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		u.Email = p.Email
	case EventAddressChanged:
		var p AddressChangedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		u.Address = p.Address
	case EventAvatarChanged:
		var p AvatarChangedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		u.Avatar = p.Avatar
	case EventDeleted:
		u.Deleted = true
	case EventPasswordChanged:
		// password hash never exposed in read model
	default:
		// unknown events are ignored for forward compatibility
	}
	return nil
}

func (u *User) MarshalSnapshot() (json.RawMessage, error) {
	return json.Marshal(u)
}

func (u *User) UnmarshalSnapshot(data json.RawMessage) error {
	return json.Unmarshal(data, u)
}

func (u *User) ToMap() map[string]any {
	return map[string]any{
		"id":         u.ID,
		"name":       u.Name,
		"email":      u.Email,
		"address":    u.Address,
		"avatar":     u.Avatar,
		"deleted":    u.Deleted,
		"version":    u.Version,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}

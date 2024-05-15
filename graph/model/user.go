package model

type User struct {
	ID             string  `json:"id"`
	Name           *string `json:"name,omitempty"`
	Email          string  `json:"email"`
	AvatarUrl      *string `json:"avatarUrl,omitempty"`
	HashedPassword string  `json:"hashedPassword"`
}

package mapper

import (
	authdto "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service/dto"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

func UserToDTO(user userdomain.User) authdto.User {
	return authdto.User{
		ID:         string(user.ID),
		Username:   user.Username,
		CreatedAt:  user.CreatedAt,
		LastSeenAt: user.LastSeenAt,
	}
}

func UserFromDTO(dto authdto.User) userdomain.User {
	return userdomain.User{
		ID:         userdomain.ID(dto.ID),
		Username:   dto.Username,
		CreatedAt:  dto.CreatedAt,
		LastSeenAt: dto.LastSeenAt,
	}
}

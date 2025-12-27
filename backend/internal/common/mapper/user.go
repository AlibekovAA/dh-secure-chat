package mapper

import (
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/dto"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

func UserToDTO(user userdomain.User) dto.User {
	return dto.User{
		ID:         string(user.ID),
		Username:   user.Username,
		CreatedAt:  user.CreatedAt,
		LastSeenAt: user.LastSeenAt,
	}
}

func UserFromDTO(dto dto.User) userdomain.User {
	return userdomain.User{
		ID:         userdomain.ID(dto.ID),
		Username:   dto.Username,
		CreatedAt:  dto.CreatedAt,
		LastSeenAt: dto.LastSeenAt,
	}
}

func UserSummaryToDTO(summary userdomain.Summary) dto.UserSummary {
	return dto.UserSummary{
		ID:        string(summary.ID),
		Username:  summary.Username,
		CreatedAt: summary.CreatedAt,
	}
}

func UserSummariesToDTO(summaries []userdomain.Summary) []dto.UserSummary {
	result := make([]dto.UserSummary, len(summaries))
	for i, s := range summaries {
		result[i] = UserSummaryToDTO(s)
	}
	return result
}

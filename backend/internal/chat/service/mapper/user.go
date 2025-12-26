package mapper

import (
	chatdto "github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/service/dto"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

func UserToDTO(user userdomain.User) chatdto.User {
	return chatdto.User{
		ID:         string(user.ID),
		Username:   user.Username,
		CreatedAt:  user.CreatedAt,
		LastSeenAt: user.LastSeenAt,
	}
}

func UserSummaryToDTO(summary userdomain.Summary) chatdto.UserSummary {
	return chatdto.UserSummary{
		ID:        string(summary.ID),
		Username:  summary.Username,
		CreatedAt: summary.CreatedAt,
	}
}

func UserSummariesToDTO(summaries []userdomain.Summary) []chatdto.UserSummary {
	result := make([]chatdto.UserSummary, len(summaries))
	for i, s := range summaries {
		result[i] = UserSummaryToDTO(s)
	}
	return result
}

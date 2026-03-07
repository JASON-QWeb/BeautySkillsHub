package handler

import (
	"strings"

	"skill-hub/internal/model"

	"github.com/gin-gonic/gin"
)

const defaultSkillAuthor = "Anonymous"

func currentUserIdentity(c *gin.Context) (uint, string, bool) {
	rawUserID, ok := c.Get("userID")
	if !ok {
		return 0, "", false
	}

	userID, ok := rawUserID.(uint)
	if !ok || userID == 0 {
		return 0, "", false
	}

	rawUsername, _ := c.Get("username")
	username, _ := rawUsername.(string)
	return userID, strings.TrimSpace(username), true
}

func optionalCurrentUserID(c *gin.Context) uint {
	rawUserID, ok := c.Get("userID")
	if !ok {
		return 0
	}
	userID, ok := rawUserID.(uint)
	if !ok {
		return 0
	}
	return userID
}

func normalizeSkillAuthor(authUsername, submittedAuthor string) string {
	if name := strings.TrimSpace(authUsername); name != "" {
		return name
	}
	if name := strings.TrimSpace(submittedAuthor); name != "" {
		return name
	}
	return defaultSkillAuthor
}

func canManageSkill(skill *model.Skill, userID uint, username string) bool {
	if skill == nil || userID == 0 {
		return false
	}

	if skill.UserID != 0 {
		return skill.UserID == userID
	}

	if strings.TrimSpace(username) == "" {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(skill.Author), strings.TrimSpace(username))
}

func canReviewSkill(skill *model.Skill, userID uint, username string) bool {
	if skill == nil || userID == 0 {
		return false
	}

	if !skill.AIApproved {
		return false
	}

	if skill.HumanReviewStatus == model.HumanReviewStatusApproved {
		return false
	}

	if canManageSkill(skill, userID, username) {
		return false
	}

	return true
}

package handler

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxContentNameLength        = 255
	maxContentDescriptionLength = 5000
	maxContentTagsLength        = 1000
	maxContentAuthorLength      = 100
	maxContentSourceURLLength   = 1024
)

type contentTextFields struct {
	Name        string
	Description string
	Tags        string
	Author      string
	SourceURL   string
}

type fieldLengthError struct {
	Field string
	Limit int
}

func (e *fieldLengthError) Error() string {
	return fmt.Sprintf("%s exceeds %d characters", e.Field, e.Limit)
}

func validateContentTextFields(fields contentTextFields) error {
	checks := []struct {
		field string
		limit int
		value string
	}{
		{field: "name", limit: maxContentNameLength, value: fields.Name},
		{field: "description", limit: maxContentDescriptionLength, value: fields.Description},
		{field: "tags", limit: maxContentTagsLength, value: fields.Tags},
		{field: "author", limit: maxContentAuthorLength, value: fields.Author},
		{field: "source_url", limit: maxContentSourceURLLength, value: fields.SourceURL},
	}

	for _, check := range checks {
		if utf8.RuneCountInString(strings.TrimSpace(check.value)) > check.limit {
			return &fieldLengthError{Field: check.field, Limit: check.limit}
		}
	}

	return nil
}

func formatSkillFieldLengthError(err *fieldLengthError) string {
	if err == nil {
		return "输入内容过长"
	}
	switch err.Field {
	case "name":
		return fmt.Sprintf("名称长度不能超过 %d 个字符", err.Limit)
	case "description":
		return fmt.Sprintf("描述长度不能超过 %d 个字符", err.Limit)
	case "tags":
		return fmt.Sprintf("标签长度不能超过 %d 个字符", err.Limit)
	case "author":
		return fmt.Sprintf("作者长度不能超过 %d 个字符", err.Limit)
	case "source_url":
		return fmt.Sprintf("来源链接长度不能超过 %d 个字符", err.Limit)
	default:
		return "输入内容过长"
	}
}

func formatResourceFieldLengthError(err *fieldLengthError) string {
	if err == nil {
		return "content exceeds maximum length"
	}
	return err.Error()
}

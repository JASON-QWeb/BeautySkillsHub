package handler

import (
	"errors"

	"skill-hub/internal/service"
)

var incrementDownloadCounter = func(svc *service.SkillService, id uint) error {
	if svc == nil {
		return errors.New("skill service not configured")
	}
	return svc.IncrementDownload(id)
}

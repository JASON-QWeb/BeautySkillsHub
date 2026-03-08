package handler

import (
	"fmt"
	"strings"
	"time"

	"skill-hub/internal/model"
)

func (h *SkillHandler) dispatchAIReviewRevision(revisionID uint) {
	if h.aiSvc == nil {
		return
	}

	key := reviewQueueKey("revision", revisionID)
	h.reviewMu.Lock()
	if _, exists := h.reviewRunning[key]; exists {
		h.reviewMu.Unlock()
		return
	}
	h.reviewRunning[key] = struct{}{}
	h.reviewMu.Unlock()

	go h.runAIReviewRevision(revisionID)
}

func (h *SkillHandler) runAIReviewRevision(revisionID uint) {
	key := reviewQueueKey("revision", revisionID)
	defer func() {
		h.reviewMu.Lock()
		delete(h.reviewRunning, key)
		h.reviewMu.Unlock()
	}()

	revision, err := h.skillSvc.GetSkillRevision(revisionID)
	if err != nil {
		return
	}
	if revision.Status != model.SkillRevisionStatusPending {
		return
	}
	if revision.AIReviewStatus == model.AIReviewStatusPassed {
		return
	}

	maxAttempts := revision.AIReviewMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if revision.AIReviewAttempts >= maxAttempts {
		revision.AIReviewStatus = model.AIReviewStatusFailedTerminal
		revision.AIFeedback = "已达最大重试次数，请重新提交更新"
		_ = h.skillSvc.UpdateSkillRevision(revision)
		return
	}

	now := time.Now()
	revision.AIReviewAttempts++
	revision.AIReviewStatus = model.AIReviewStatusRunning
	revision.AIReviewPhase = model.AIReviewPhaseSecurity
	revision.AIReviewStartedAt = &now
	revision.AIReviewCompletedAt = nil
	revision.AIReviewDetails = ""
	revision.AIFeedback = "准备审核更新内容..."
	revision.AIApproved = false
	if err := h.skillSvc.UpdateSkillRevision(revision); err != nil {
		return
	}

	targets, err := h.collectReviewTargetsForFile(revision.FilePath)
	if err != nil {
		h.finishRevisionReviewAsError(revision, maxAttempts, fmt.Sprintf("准备审核文件失败: %v", err))
		return
	}

	progress := newReviewProgress(targets)
	revision.AIReviewDetails = encodeReviewProgress(progress)
	revision.AIFeedback = fmt.Sprintf("共发现 %d 个待审核文件", progress.TotalFiles)
	if err := h.skillSvc.UpdateSkillRevision(revision); err != nil {
		return
	}

	failedFiles := make([]string, 0)
	functionalSummaries := make([]string, 0, len(targets))
	for i := range targets {
		target := targets[i]
		progress.CurrentFile = target.Path
		progress.Files[i].Status = reviewFileStatusRunning
		progress.Files[i].Message = ""

		revision.AIReviewPhase = model.AIReviewPhaseSecurity
		revision.AIFeedback = fmt.Sprintf("安全性审核中：%s", target.Path)
		revision.AIReviewDetails = encodeReviewProgress(progress)
		if err := h.skillSvc.UpdateSkillRevision(revision); err != nil {
			return
		}

		content, readErr := readReviewContent(target.LocalPath, reviewContentLimitBytes)
		if readErr != nil {
			progress.Files[i].Status = reviewFileStatusFailed
			progress.Files[i].Message = fmt.Sprintf("读取失败: %v", readErr)
			progress.CompletedFiles++
			revision.AIReviewDetails = encodeReviewProgress(progress)
			h.finishRevisionReviewAsError(revision, maxAttempts, fmt.Sprintf("读取文件失败（%s）: %v", target.Path, readErr))
			return
		}

		localFindings := detectLocalRiskFindings(content)

		revision.AIReviewPhase = model.AIReviewPhaseFunctional
		revision.AIFeedback = fmt.Sprintf("功能性审核中：%s", target.Path)
		revision.AIReviewDetails = encodeReviewProgress(progress)
		if err := h.skillSvc.UpdateSkillRevision(revision); err != nil {
			return
		}

		reviewResult, aiErr := h.aiSvc.ReviewSkill(
			revision.Name,
			revision.ResourceType,
			fmt.Sprintf("%s\n审查文件: %s\n文件类型: %s", revision.Description, target.Path, target.Kind),
			content,
		)
		if aiErr != nil {
			progress.Files[i].Status = reviewFileStatusFailed
			progress.Files[i].Message = "AI 调用失败"
			progress.CompletedFiles++
			revision.AIReviewDetails = encodeReviewProgress(progress)
			h.finishRevisionReviewAsError(revision, maxAttempts, fmt.Sprintf("AI 审核失败（%s）: %v", target.Path, aiErr))
			return
		}
		if summary := extractFunctionalSummary(reviewResult.FuncSummary, reviewResult.AIDescription, reviewResult.Feedback); summary != "" {
			functionalSummaries = append(functionalSummaries, summary)
		}

		filePassed := true
		messages := make([]string, 0, 2)
		if len(localFindings) > 0 {
			filePassed = false
			messages = append(messages, "命中风险规则: "+strings.Join(localFindings, "；"))
		}
		if !reviewResult.Approved {
			filePassed = false
			if msg := strings.TrimSpace(reviewResult.Feedback); msg != "" {
				messages = append(messages, truncateReviewMessage(msg, 160))
			} else {
				messages = append(messages, "AI 判定存在安全或功能风险")
			}
		}

		if filePassed {
			progress.Files[i].Status = reviewFileStatusPassed
			if msg := strings.TrimSpace(reviewResult.Feedback); msg != "" {
				progress.Files[i].Message = truncateReviewMessage(msg, 120)
			}
		} else {
			progress.Files[i].Status = reviewFileStatusFailed
			progress.Files[i].Message = truncateReviewMessage(strings.Join(messages, "；"), 220)
			failedFiles = append(failedFiles, target.Path)
		}
		progress.CompletedFiles++
		revision.AIReviewDetails = encodeReviewProgress(progress)
		if err := h.skillSvc.UpdateSkillRevision(revision); err != nil {
			return
		}
	}

	revision.AIReviewPhase = model.AIReviewPhaseFinalizing
	revision.AIFeedback = "结果归档中..."
	revision.AIReviewDetails = encodeReviewProgress(progress)
	_ = h.skillSvc.UpdateSkillRevision(revision)

	doneAt := time.Now()
	progress.CurrentFile = ""
	revision.AIReviewCompletedAt = &doneAt
	revision.AIReviewPhase = model.AIReviewPhaseDone
	revision.AIReviewDetails = encodeReviewProgress(progress)

	if len(failedFiles) == 0 {
		revision.AIApproved = true
		revision.AIReviewStatus = model.AIReviewStatusPassed
		revision.AIFeedback = fmt.Sprintf("AI 审核通过，已检查 %d 个关键文件", len(targets))
		functionalSummary := truncateReviewMessage(buildFunctionalReviewSummary(revision.Name, functionalSummaries), 72)
		functionalContext := truncateReviewMessage(buildFunctionalContextLine(revision.Name, revision.Description, len(targets)), 72)
		revision.AIDescription = fmt.Sprintf("功能概述: %s\n功能亮点: %s", functionalSummary, functionalContext)
	} else {
		revision.AIApproved = false
		if revision.AIReviewAttempts >= maxAttempts {
			revision.AIReviewStatus = model.AIReviewStatusFailedTerminal
			revision.Status = model.SkillRevisionStatusRejected
		} else {
			revision.AIReviewStatus = model.AIReviewStatusFailedRetry
		}
		revision.AIFeedback = fmt.Sprintf("发现 %d 个风险文件：%s", len(failedFiles), summarizeFileList(failedFiles, 3))
		functionalSummary := truncateReviewMessage(buildFunctionalReviewSummary(revision.Name, functionalSummaries), 72)
		improvement := truncateReviewMessage(fmt.Sprintf("当前检测到 %d 个待修复文件，建议修复后重新提交审核。", len(failedFiles)), 72)
		revision.AIDescription = fmt.Sprintf("功能概述: %s\n改进建议: %s", functionalSummary, improvement)
	}

	_ = h.skillSvc.UpdateSkillRevision(revision)
}

func (h *SkillHandler) finishRevisionReviewAsError(revision *model.SkillRevision, maxAttempts int, message string) {
	doneAt := time.Now()
	revision.AIApproved = false
	revision.AIReviewCompletedAt = &doneAt
	revision.AIReviewPhase = model.AIReviewPhaseDone
	if revision.AIReviewAttempts >= maxAttempts {
		revision.AIReviewStatus = model.AIReviewStatusFailedTerminal
		revision.Status = model.SkillRevisionStatusRejected
	} else {
		revision.AIReviewStatus = model.AIReviewStatusFailedRetry
	}
	revision.AIFeedback = strings.TrimSpace(message)
	if revision.AIFeedback == "" {
		revision.AIFeedback = "AI 审核失败，请稍后重试"
	}
	if strings.TrimSpace(revision.AIDescription) == "" {
		revision.AIDescription = "功能概述: 本次 AI 审核未完成。\n改进建议: 请稍后重试或重新提交更新。"
	}
	_ = h.skillSvc.UpdateSkillRevision(revision)
}

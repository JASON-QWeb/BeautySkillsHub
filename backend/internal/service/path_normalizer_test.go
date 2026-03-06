package service

import "testing"

func TestBuildSkillRepoPath_UsesSlugFolderAndOriginalFile(t *testing.T) {
	dir, file := BuildSkillRepoPath("skills", "tools", "My Skill 101", "script.sh")

	if dir != "skills/tools/my-skill-101" {
		t.Fatalf("expected normalized dir path, got %q", dir)
	}
	if file != "skills/tools/my-skill-101/script.sh" {
		t.Fatalf("expected normalized file path, got %q", file)
	}
}

func TestBuildSkillRepoPath_UsesFallbackForEmptyTitle(t *testing.T) {
	dir, file := BuildSkillRepoPath("skills", "skill", "   ", "note.md")

	if dir != "skills/skill/untitled-skill" {
		t.Fatalf("expected fallback title folder, got %q", dir)
	}
	if file != "skills/skill/untitled-skill/note.md" {
		t.Fatalf("expected file under fallback title folder, got %q", file)
	}
}

func TestBuildSkillRepoPath_SanitizesFilenameAndResourceType(t *testing.T) {
	dir, file := BuildSkillRepoPath("skills", "../weird", "Hello", "../../a\\b/../evil.txt")

	if dir != "skills/skill/hello" {
		t.Fatalf("expected invalid resource type fallback to skill, got %q", dir)
	}
	if file != "skills/skill/hello/evil.txt" {
		t.Fatalf("expected dangerous filename cleaned, got %q", file)
	}
}

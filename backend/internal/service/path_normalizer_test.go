package service

import "testing"

func TestBuildSkillRepoPath_UsesSlugFolderAndOriginalFile(t *testing.T) {
	dir, file := BuildSkillRepoPath("skills", "tools", "My Skill 101", "script.sh")

	if dir != "skills/my-skill-101" {
		t.Fatalf("expected normalized dir path, got %q", dir)
	}
	if file != "skills/my-skill-101/script.sh" {
		t.Fatalf("expected normalized file path, got %q", file)
	}
}

func TestBuildSkillRepoPath_UsesFallbackForEmptyTitle(t *testing.T) {
	dir, file := BuildSkillRepoPath("skills", "skill", "   ", "note.md")

	if dir != "skills/untitled-skill" {
		t.Fatalf("expected fallback title folder, got %q", dir)
	}
	if file != "skills/untitled-skill/note.md" {
		t.Fatalf("expected file under fallback title folder, got %q", file)
	}
}

func TestBuildSkillRepoPath_SanitizesFilenameAndResourceType(t *testing.T) {
	dir, file := BuildSkillRepoPath("skills", "../weird", "Hello", "../../a\\b/../evil.txt")

	if dir != "skills/hello" {
		t.Fatalf("expected title folder under base dir, got %q", dir)
	}
	if file != "skills/hello/evil.txt" {
		t.Fatalf("expected dangerous filename cleaned, got %q", file)
	}
}

func TestBuildSkillRepoPath_PreservesUnicodeTitle(t *testing.T) {
	dir, file := BuildSkillRepoPath("skills", "skill", "中文 技能", "README.md")
	if dir != "skills/中文-技能" {
		t.Fatalf("expected unicode folder name, got %q", dir)
	}
	if file != "skills/中文-技能/README.md" {
		t.Fatalf("expected unicode file path, got %q", file)
	}
}

func TestNormalizeRepoRelativePath(t *testing.T) {
	t.Run("keeps nested relative path", func(t *testing.T) {
		got, err := NormalizeRepoRelativePath("src/cli/main.go")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != "src/cli/main.go" {
			t.Fatalf("unexpected path %q", got)
		}
	})

	t.Run("rejects traversal", func(t *testing.T) {
		_, err := NormalizeRepoRelativePath("../etc/passwd")
		if err == nil {
			t.Fatal("expected traversal path to be rejected")
		}
	})

	t.Run("rejects absolute path", func(t *testing.T) {
		_, err := NormalizeRepoRelativePath("/root/.ssh/id_rsa")
		if err == nil {
			t.Fatal("expected absolute path to be rejected")
		}
	})
}

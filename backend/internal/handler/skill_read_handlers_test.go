package handler

import "testing"

func TestResolveInstallRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repo     string
		expected string
	}{
		{
			name:     "owner and repo name",
			owner:    "JASON-QWeb",
			repo:     "agent-skills",
			expected: "https://github.com/JASON-QWeb/agent-skills",
		},
		{
			name:     "repo includes owner slash repo",
			owner:    "",
			repo:     "JASON-QWeb/agent-skills",
			expected: "https://github.com/JASON-QWeb/agent-skills",
		},
		{
			name:     "repo is full github url",
			owner:    "",
			repo:     "https://github.com/JASON-QWeb/agent-skills.git",
			expected: "https://github.com/JASON-QWeb/agent-skills",
		},
		{
			name:     "fallback when missing",
			owner:    "",
			repo:     "",
			expected: "https://github.com/skillshub/community",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveInstallRepoURL(tt.owner, tt.repo)
			if got != tt.expected {
				t.Fatalf("resolveInstallRepoURL(%q, %q)=%q, want %q", tt.owner, tt.repo, got, tt.expected)
			}
		})
	}
}

func TestSanitizeInstallBaseDir(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "normal", input: "skills", expected: "skills"},
		{name: "trim and lower", input: "  SKILLS  ", expected: "skills"},
		{name: "strip invalid chars", input: " ../Skills@V2 ", expected: "skillsv2"},
		{name: "fallback", input: "", expected: "skills"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeInstallBaseDir(tt.input)
			if got != tt.expected {
				t.Fatalf("sanitizeInstallBaseDir(%q)=%q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

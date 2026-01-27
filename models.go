package main

import (
	"regexp"
	"strings"
	"time"
)

type CommandCategory string

const (
	CategoryVCS         CommandCategory = "version-control"
	CategoryBuild       CommandCategory = "build"
	CategoryFileOps     CommandCategory = "file-operations"
	CategoryNavigation  CommandCategory = "navigation"
	CategoryDevTools    CommandCategory = "dev-tools"
	CategorySystemAdmin CommandCategory = "system-admin"
	CategoryNetwork     CommandCategory = "network"
	CategoryContainers  CommandCategory = "containers"
	CategoryDatabase    CommandCategory = "database"
	CategoryEditor      CommandCategory = "editor"
	CategorySearch      CommandCategory = "search"
	CategoryPackage     CommandCategory = "package-manager"
	CategoryOther       CommandCategory = "other"
)

type HistoryEntry struct {
	ID             int             `json:"id"`
	Timestamp      time.Time       `json:"timestamp"`
	Duration       int             `json:"duration"`
	Command        string          `json:"command"`
	Directory      string          `json:"directory"`
	Category       CommandCategory `json:"category"`
	BaseCommand    string          `json:"base_command"`
	SessionID      int             `json:"session_id"`
}

type Session struct {
	ID           int            `json:"id"`
	StartTime    time.Time      `json:"start_time"`
	EndTime      time.Time      `json:"end_time"`
	Duration     time.Duration  `json:"duration"`
	Commands     []HistoryEntry `json:"commands"`
	Directories  []string       `json:"directories"`
	Categories   map[CommandCategory]int `json:"categories"`
	Description  string         `json:"description"`
}

type CommandPattern struct {
	Command      string                  `json:"command"`
	Count        int                     `json:"count"`
	CoOccurrence map[string]int          `json:"co_occurrence"`
	Categories   map[CommandCategory]int `json:"categories"`
}

var categoryPatterns = map[CommandCategory]*regexp.Regexp{
	CategoryVCS:         regexp.MustCompile(`^(git|hg|svn|bzr|cvs)\s`),
	CategoryBuild:       regexp.MustCompile(`^(make|cmake|cargo|npm|yarn|pnpm|gradle|mvn|ant|bazel|go build|go test|gcc|g\+\+|clang|rustc|javac)\s`),
	CategoryFileOps:     regexp.MustCompile(`^(cp|mv|rm|mkdir|rmdir|touch|chmod|chown|ln|cat|head|tail|less|more|dd|rsync|scp)\s`),
	CategoryNavigation:  regexp.MustCompile(`^(cd|ls|pwd|tree|find|locate|which|whereis)\s`),
	CategoryDevTools:    regexp.MustCompile(`^(vim|nvim|emacs|nano|code|subl|idea|pycharm|gdb|lldb|valgrind|strace|ltrace)\s`),
	CategorySystemAdmin: regexp.MustCompile(`^(sudo|su|systemctl|service|kill|killall|ps|top|htop|free|df|du|mount|umount|lsof|netstat|ss|iptables|ufw|systemd)\s`),
	CategoryNetwork:     regexp.MustCompile(`^(curl|wget|ssh|scp|rsync|ping|traceroute|nslookup|dig|host|telnet|nc|netcat|ftp|sftp)\s`),
	CategoryContainers:  regexp.MustCompile(`^(docker|podman|kubectl|k|helm|minikube|kind|k3s|nerdctl|containerd)\s`),
	CategoryDatabase:    regexp.MustCompile(`^(psql|mysql|sqlite3|mongo|redis-cli|mongosh|clickhouse-client)\s`),
	CategoryEditor:      regexp.MustCompile(`^(vim|nvim|emacs|nano|vi|ed|joe|pico)\s`),
	CategorySearch:      regexp.MustCompile(`^(grep|egrep|fgrep|ag|rg|ack|find.*-name|locate)\s`),
	CategoryPackage:     regexp.MustCompile(`^(apt|apt-get|yum|dnf|pacman|brew|pip|pip3|npm|yarn|cargo|gem|composer)\s`),
}

func CategorizeCommand(cmd string) CommandCategory {
	cmd = strings.TrimSpace(cmd)
	for category, pattern := range categoryPatterns {
		if pattern.MatchString(cmd) {
			return category
		}
	}
	return CategoryOther
}

func GetBaseCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) > 0 {
		return parts[0]
	}
	return cmd
}

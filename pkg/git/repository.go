package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Repository manages git repository operations
type Repository struct {
	logger      *zap.Logger
	repoURL     string
	localPath   string
	branch      string
	workspaceDir string
	gitUsername string
	gitEmail    string
}

// RepositoryConfig contains configuration for git repository
type RepositoryConfig struct {
	URL          string `json:"url"`
	Branch       string `json:"branch"`
	WorkspaceDir string `json:"workspace_dir"`
	GitUsername  string `json:"git_username"`
	GitEmail     string `json:"git_email"`
}

func NewRepository(config RepositoryConfig, logger *zap.Logger) *Repository {
	// Extract repository name from URL
	repoName := extractRepoName(config.URL)
	localPath := filepath.Join(config.WorkspaceDir, repoName)

	return &Repository{
		logger:       logger,
		repoURL:      config.URL,
		localPath:    localPath,
		branch:       config.Branch,
		workspaceDir: config.WorkspaceDir,
		gitUsername:  config.GitUsername,
		gitEmail:     config.GitEmail,
	}
}

// Initialize ensures the repository is cloned and up to date
func (r *Repository) Initialize() error {
	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(r.workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Check if directory exists but is not a valid git repository
	if r.directoryExists() && !r.exists() {
		r.logger.Warn("Directory exists but is not a valid git repository, cleaning up", 
			zap.String("path", r.localPath))
		if err := r.cleanup(); err != nil {
			return fmt.Errorf("failed to cleanup invalid repository: %w", err)
		}
	}

	// Check if repository exists locally and is valid
	if r.exists() {
		r.logger.Info("Repository exists locally, pulling latest changes", 
			zap.String("path", r.localPath))
		
		// Try to pull, if it fails, cleanup and re-clone
		if err := r.pull(); err != nil {
			r.logger.Warn("Failed to pull repository, cleaning up and re-cloning", 
				zap.String("error", err.Error()))
			if cleanupErr := r.cleanup(); cleanupErr != nil {
				return fmt.Errorf("failed to cleanup repository after pull failure: %w", cleanupErr)
			}
			return r.clone()
		}
		return nil
	}

	// Clone the repository
	r.logger.Info("Repository not found locally, cloning", 
		zap.String("url", r.repoURL),
		zap.String("path", r.localPath))
	return r.clone()
}

// directoryExists checks if the local path directory exists
func (r *Repository) directoryExists() bool {
	if _, err := os.Stat(r.localPath); err != nil {
		return false
	}
	return true
}

// exists checks if the repository exists locally and is a valid git repository
func (r *Repository) exists() bool {
	gitDir := filepath.Join(r.localPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return false
	}
	
	// Additional check: verify it's a valid git repository
	return r.isValidGitRepository()
}

// isValidGitRepository checks if the directory is a valid git repository
func (r *Repository) isValidGitRepository() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = r.localPath
	err := cmd.Run()
	return err == nil
}

// cleanup removes the local repository directory
func (r *Repository) cleanup() error {
	if r.directoryExists() {
		r.logger.Info("Cleaning up repository directory", zap.String("path", r.localPath))
		if err := os.RemoveAll(r.localPath); err != nil {
			return fmt.Errorf("failed to remove directory: %w", err)
		}
	}
	return nil
}

// clone clones the repository from remote
func (r *Repository) clone() error {
	// Extract just the repo name for git clone command
	repoName := extractRepoName(r.repoURL)
	cmd := exec.Command("git", "clone", "-b", r.branch, r.repoURL, repoName)
	cmd.Dir = r.workspaceDir
	
	// Set up environment for SSH operations
	if r.isSSHURL(r.repoURL) {
		r.setupSSHEnvironment(cmd)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repository: %s, output: %s", err, string(output))
	}

	r.logger.Info("Repository cloned successfully", 
		zap.String("url", r.repoURL),
		zap.String("branch", r.branch),
		zap.String("path", r.localPath))
	
	return nil
}

// pull pulls the latest changes from remote
func (r *Repository) pull() error {
	// First, checkout the target branch
	if err := r.checkoutBranch(r.branch); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	cmd := exec.Command("git", "pull", "origin", r.branch)
	cmd.Dir = r.localPath
	
	// Set up environment for SSH operations
	if r.isSSHURL(r.repoURL) {
		r.setupSSHEnvironment(cmd)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull repository: %s, output: %s", err, string(output))
	}

	r.logger.Info("Repository pulled successfully", 
		zap.String("branch", r.branch),
		zap.String("output", string(output)))
	
	return nil
}

// checkoutBranch switches to the specified branch
func (r *Repository) checkoutBranch(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = r.localPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If branch doesn't exist locally, try to create it from remote
		if strings.Contains(string(output), "did not match any file") {
			return r.createBranchFromRemote(branch)
		}
		return fmt.Errorf("failed to checkout branch: %s, output: %s", err, string(output))
	}

	return nil
}

// createBranchFromRemote creates a local branch tracking the remote branch
func (r *Repository) createBranchFromRemote(branch string) error {
	// Fetch latest refs
	fetchCmd := exec.Command("git", "fetch", "origin")
	fetchCmd.Dir = r.localPath
	
	// Set up environment for SSH operations
	if r.isSSHURL(r.repoURL) {
		r.setupSSHEnvironment(fetchCmd)
	}
	
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch from origin: %w", err)
	}

	// Create and checkout branch from remote
	cmd := exec.Command("git", "checkout", "-b", branch, "origin/"+branch)
	cmd.Dir = r.localPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create branch from remote: %s, output: %s", err, string(output))
	}

	r.logger.Info("Created local branch from remote", 
		zap.String("branch", branch))
	
	return nil
}

// Add stages files for commit
func (r *Repository) Add(files ...string) error {
	if len(files) == 0 {
		files = []string{"."}
	}

	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = r.localPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add files: %s, output: %s", err, string(output))
	}

	r.logger.Debug("Files added to git", 
		zap.Strings("files", files))
	
	return nil
}

// ConfigureGitUser sets up git user configuration for the repository
func (r *Repository) ConfigureGitUser() error {
	if r.gitUsername == "" || r.gitEmail == "" {
		r.logger.Warn("Git username or email not configured, skipping git user setup",
			zap.String("username", r.gitUsername),
			zap.String("email", r.gitEmail))
		return nil
	}

	// Set git user name
	cmd := exec.Command("git", "config", "user.name", r.gitUsername)
	cmd.Dir = r.localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set git user name: %s, output: %s", err, string(output))
	}

	// Set git user email
	cmd = exec.Command("git", "config", "user.email", r.gitEmail)
	cmd.Dir = r.localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set git user email: %s, output: %s", err, string(output))
	}

	r.logger.Info("Git user configured successfully",
		zap.String("username", r.gitUsername),
		zap.String("email", r.gitEmail))

	return nil
}

// Commit creates a commit with the given message
func (r *Repository) Commit(message string) error {
	// Configure git user before committing
	if err := r.ConfigureGitUser(); err != nil {
		return fmt.Errorf("failed to configure git user: %w", err)
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = r.localPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if there are no changes to commit
		if strings.Contains(string(output), "nothing to commit") {
			r.logger.Info("No changes to commit")
			return nil
		}
		return fmt.Errorf("failed to commit: %s, output: %s", err, string(output))
	}

	r.logger.Info("Committed changes", 
		zap.String("message", message))
	
	return nil
}

// Push pushes commits to remote
func (r *Repository) Push() error {
	cmd := exec.Command("git", "push", "origin", r.branch)
	cmd.Dir = r.localPath
	
	// Set up environment for SSH operations
	if r.isSSHURL(r.repoURL) {
		r.setupSSHEnvironment(cmd)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push: %s, output: %s", err, string(output))
	}

	r.logger.Info("Pushed to remote", 
		zap.String("branch", r.branch),
		zap.String("output", string(output)))
	
	return nil
}

// GetLastCommitHash returns the hash of the last commit
func (r *Repository) GetLastCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = r.localPath
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetStatus returns the git status
func (r *Repository) GetStatus() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = r.localPath
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	return string(output), nil
}

// HasChanges checks if there are any uncommitted changes
func (r *Repository) HasChanges() (bool, error) {
	status, err := r.GetStatus()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(status) != "", nil
}

// GetLocalPath returns the local path of the repository
func (r *Repository) GetLocalPath() string {
	return r.localPath
}

// GetBranch returns the current branch
func (r *Repository) GetBranch() string {
	return r.branch
}

// CreateFile creates a file in the repository
func (r *Repository) CreateFile(relativePath string, content []byte) error {
	fullPath := filepath.Join(r.localPath, relativePath)
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	r.logger.Debug("File created in repository", 
		zap.String("path", relativePath))
	
	return nil
}

// FileExists checks if a file exists in the repository
func (r *Repository) FileExists(relativePath string) bool {
	fullPath := filepath.Join(r.localPath, relativePath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// Helper function to extract repository name from URL
func extractRepoName(url string) string {
	// Remove .git suffix if present
	if strings.HasSuffix(url, ".git") {
		url = strings.TrimSuffix(url, ".git")
	}
	
	// Handle SSH URLs (git@github.com:user/repo)
	if strings.Contains(url, ":") && strings.Contains(url, "@") {
		// Split by colon and get the last part
		parts := strings.Split(url, ":")
		if len(parts) > 1 {
			path := parts[len(parts)-1]
			// Get the repository name (last part after /)
			pathParts := strings.Split(path, "/")
			if len(pathParts) > 0 {
				return pathParts[len(pathParts)-1]
			}
		}
	}
	
	// Get the last part of the URL for HTTPS URLs
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	
	return "repo"
}

// isSSHURL checks if the given URL is an SSH URL
func (r *Repository) isSSHURL(url string) bool {
	return strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://")
}

// setupSSHEnvironment sets up the SSH environment for git commands
func (r *Repository) setupSSHEnvironment(cmd *exec.Cmd) {
	// Set up SSH options to handle host key verification
	// This will automatically accept unknown host keys (be careful in production)
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	
	// Set Git SSH command to use SSH with specific options
	sshCommand := "ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
	cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND="+sshCommand)
	
	r.logger.Debug("SSH environment configured for git command",
		zap.String("ssh_command", sshCommand))
}
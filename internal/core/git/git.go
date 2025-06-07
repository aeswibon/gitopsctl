package git

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/plumbing/transport"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"go.uber.org/zap"
)

// CloneOrPull performs a Git clone if the target directory doesn't contain a valid Git repository.
// If the repository already exists, it performs a Git pull to fetch the latest changes.
//
// Returns the HEAD commit hash after the operation.
func CloneOrPull(logger *zap.Logger, repoURL, branch, targetDir string) (string, error) {
	var repo *gogit.Repository
	var err error

	// First, try to open the directory as an existing Git repository.
	repo, err = gogit.PlainOpen(targetDir)
	if err != nil {
		// If opening fails, check if it's because the repository does not exist at the path.
		// This can happen if the directory is empty or not a Git repo.
		if err == gogit.ErrRepositoryNotExists || strings.Contains(err.Error(), "git repository not found") {
			// Directory exists but is not a Git repo, or path does not exist. Clone it.
			logger.Info("Cloning repository",
				zap.String("repoURL", repoURL),
				zap.String("branch", branch),
				zap.String("targetDir", targetDir),
			)
			repo, err = gogit.PlainClone(targetDir, false, &gogit.CloneOptions{
				URL:           repoURL,
				ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
				SingleBranch:  true,
				Depth:         1, // Only clone the latest commit for efficiency
				Progress:      os.Stdout,
				Auth:          setupAuth(repoURL), // Handles SSH agent/keys
			})
			if err != nil {
				return "", fmt.Errorf("failed to clone repository %s: %w", repoURL, err)
			}
		} else {
			// Another error occurred while trying to open the repository.
			return "", fmt.Errorf("failed to open existing repository %s: %w", targetDir, err)
		}
	} else {
		// Repository already exists and was successfully opened, perform a pull.
		logger.Debug("Pulling repository",
			zap.String("repoURL", repoURL),
			zap.String("branch", branch),
			zap.String("targetDir", targetDir),
		)
		worktree, err := repo.Worktree()
		if err != nil {
			return "", fmt.Errorf("failed to get worktree for %s: %w", targetDir, err)
		}

		err = worktree.Pull(&gogit.PullOptions{
			RemoteName:    "origin",
			ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
			SingleBranch:  true,
			Progress:      os.Stdout,
			Auth:          setupAuth(repoURL), // Handles SSH agent/keys
		})
		if err != nil {
			if err == gogit.NoErrAlreadyUpToDate {
				logger.Debug("Repository already up-to-date", zap.String("repoURL", repoURL))
			} else {
				return "", fmt.Errorf("failed to pull repository %s: %w", repoURL, err)
			}
		}
	}

	// Get the HEAD commit hash after either clone or pull operation.
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD after Git operation: %w", err)
	}
	return head.Hash().String(), nil
}

// GetLatestCommitHash retrieves the HEAD commit hash of a local Git repository.
//
// This function opens the repository at the specified path and reads the current HEAD reference.
func GetLatestCommitHash(logger *zap.Logger, repoPath string) (string, error) {
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository %s: %w", repoPath, err)
	}
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD for repository %s: %w", repoPath, err)
	}
	return head.Hash().String(), nil
}

// setupAuth provides authentication for Git operations.
//
// For SSH-based repositories, it attempts to use the SSH agent or default SSH keys.
// For HTTPS-based repositories, it currently supports public repositories without authentication.
// In production, this function could be extended to handle tokens, username/password, or specific key files.
func setupAuth(repoURL string) transport.AuthMethod {
	if strings.HasPrefix(repoURL, "git@") || strings.HasPrefix(repoURL, "ssh://") {
		// Try to use SSH agent or default SSH keys (~/.ssh/id_rsa)
		sshAuth, err := ssh.NewSSHAgentAuth("") // Empty string uses default agent/keys
		if err != nil {
			zap.L().Warn("Could not use SSH agent for Git authentication, falling back to public repos", zap.Error(err))
			return nil // Fallback to no authentication (will work for public repos)
		}
		return sshAuth
	}
	// For HTTPS, no explicit AuthMethod for public repos.
	// For private HTTPS repos, you'd need http.BasicAuth or similar.
	return nil
}

// CleanUpRepo deletes the local repository directory.
//
// This function is used to clean up temporary directories created for Git operations.
func CleanUpRepo(logger *zap.Logger, repoDir string) error {
	logger.Info("Cleaning up local repository directory", zap.String("dir", repoDir))
	if err := os.RemoveAll(repoDir); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", repoDir, err)
	}
	return nil
}

// CreateTempRepoDir creates a temporary directory for cloning a repository.
//
// The directory is created with a unique name to ensure isolation between different Git operations.
func CreateTempRepoDir() (string, error) {
	tmpDir, err := os.MkdirTemp("", "gitopsctl-repo-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	return tmpDir, nil
}

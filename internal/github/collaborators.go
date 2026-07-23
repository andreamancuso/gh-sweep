package github

import (
	"fmt"
	"time"
)

// Collaborator represents a repository collaborator
type Collaborator struct {
	Login      string
	Permission string
	Repository string
}

type collaboratorResponse struct {
	Login       string `json:"login"`
	Permissions struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions"`
}

// ListCollaborators lists all collaborators for a repository
func (c *Client) ListCollaborators(owner, repo string) ([]Collaborator, error) {
	var response []collaboratorResponse
	path := apiPath("repos", owner, repo, "collaborators")

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list collaborators: %w", err)
	}

	collaborators := make([]Collaborator, len(response))
	for i, cr := range response {
		permission := "read"
		if cr.Permissions.Admin {
			permission = "admin"
		} else if cr.Permissions.Push {
			permission = "write"
		}

		collaborators[i] = Collaborator{
			Login:      cr.Login,
			Permission: permission,
			Repository: repoFullName(owner, repo),
		}
	}

	return collaborators, nil
}

// CollaboratorGrant represents a time-boxed access grant
type CollaboratorGrant struct {
	User       string
	Repository string
	Permission string
	GrantedBy  string
	GrantedAt  time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}

// AddCollaborator adds a collaborator to a repository
func (c *Client) AddCollaborator(owner, repo, username, permission string) error {
	body := map[string]string{
		"permission": permission,
	}

	path := apiPath("repos", owner, repo, "collaborators", username)

	if err := c.Put(path, body, nil); err != nil {
		return fmt.Errorf("failed to add collaborator: %w", err)
	}

	return nil
}

// RemoveCollaborator removes a collaborator from a repository
func (c *Client) RemoveCollaborator(owner, repo, username string) error {
	path := apiPath("repos", owner, repo, "collaborators", username)

	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to remove collaborator: %w", err)
	}

	return nil
}

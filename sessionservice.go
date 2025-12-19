package main

import (
	"fmt"
	"sort"

	"term/database"
)

type SessionService struct {
	db *database.DB
}

// NewSessionService creates a new session service
func NewSessionService(db *database.DB) *SessionService {
	return &SessionService{db: db}
}

// GetAllSessions retrieves all session nodes
func (s *SessionService) GetAllSessions() ([]database.SessionNode, error) {
	return s.db.GetAllSessions()
}

// GetSession retrieves a single session by ID
func (s *SessionService) GetSession(id string) (*database.SessionNode, error) {
	return s.db.GetSession(id)
}

// CreateSession creates a new session node
func (s *SessionService) CreateSession(session database.SessionNode) error {
	return s.db.CreateSession(&session)
}

// UpdateSession updates an existing session
func (s *SessionService) UpdateSession(session database.SessionNode) error {
	return s.db.UpdateSession(&session)
}

// DeleteSession deletes a session
func (s *SessionService) DeleteSession(id string, cascade bool) error {
	return s.db.DeleteSession(id, cascade)
}

// GetSessionConfig retrieves all direct configs for a session (not inherited)
func (s *SessionService) GetSessionConfig(sessionID string) (map[string]string, error) {
	return s.db.GetSessionConfigs(sessionID)
}

// GetEffectiveConfig gets the effective configuration with inheritance
func (s *SessionService) GetEffectiveConfig(sessionID string) (map[string]string, error) {
	return s.db.GetEffectiveConfig(sessionID)
}

// SetSessionConfig sets a config value for a session
func (s *SessionService) SetSessionConfig(sessionID, key, value, valueType string) error {
	return s.db.SetSessionConfig(sessionID, key, value, valueType)
}

// DeleteSessionConfig deletes a config key
func (s *SessionService) DeleteSessionConfig(sessionID, key string) error {
	return s.db.DeleteSessionConfig(sessionID, key)
}

// GetSessionTree builds a hierarchical tree structure from flat session list
func (s *SessionService) GetSessionTree() ([]TreeNode, error) {
	sessions, err := s.db.GetAllSessions()
	if err != nil {
		return nil, err
	}

	// Build a map for quick lookup with pointer children
	nodeMap := make(map[string]*TreeNodePtr)
	for i := range sessions {
		session := &sessions[i]
		nodeMap[session.ID] = &TreeNodePtr{
			Session:  *session,
			Children: []*TreeNodePtr{},
		}
	}

	// Build the tree with pointers
	var rootNodePtrs []*TreeNodePtr
	for _, node := range nodeMap {
		if node.Session.ParentID == nil {
			rootNodePtrs = append(rootNodePtrs, node)
		} else {
			parent, exists := nodeMap[*node.Session.ParentID]
			if exists {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	// Convert pointer tree to value tree (deep copy)
	var rootNodes []TreeNode
	for _, ptr := range rootNodePtrs {
		rootNodes = append(rootNodes, convertTreeNode(ptr))
	}

	// Sort children by position at all levels
	sortTreeByPosition(&rootNodes)

	// Log the tree structure
	fmt.Printf("BACKEND GetSessionTree: %d root nodes\n", len(rootNodes))
	for i, root := range rootNodes {
		fmt.Printf("  Root[%d]: %s (pos=%d, children=%d)\n", i, root.Session.ID, root.Session.Position, len(root.Children))
		for j, child := range root.Children {
			fmt.Printf("    Child[%d]: %s (parent=%s, pos=%d)\n", j, child.Session.ID, *child.Session.ParentID, child.Session.Position)
		}
	}

	return rootNodes, nil
}

// TreeNodePtr is a temporary structure for building the tree with pointers
type TreeNodePtr struct {
	Session  database.SessionNode
	Children []*TreeNodePtr
}

// convertTreeNode recursively converts pointer tree to value tree
func convertTreeNode(ptr *TreeNodePtr) TreeNode {
	node := TreeNode{
		Session:  ptr.Session,
		Children: make([]TreeNode, len(ptr.Children)),
	}
	for i, childPtr := range ptr.Children {
		node.Children[i] = convertTreeNode(childPtr)
	}
	return node
}

// sortTreeByPosition recursively sorts tree nodes by position
func sortTreeByPosition(nodes *[]TreeNode) {
	if nodes == nil || len(*nodes) == 0 {
		return
	}

	// Sort current level by position
	sort.Slice(*nodes, func(i, j int) bool {
		return (*nodes)[i].Session.Position < (*nodes)[j].Session.Position
	})

	// Recursively sort children
	for i := range *nodes {
		sortTreeByPosition(&(*nodes)[i].Children)
	}
}

// TreeNode represents a node in the hierarchical session tree
type TreeNode struct {
	Session  database.SessionNode `json:"session"`
	Children []TreeNode           `json:"children"`
}

// DuplicateSession creates a copy of a session with a new ID
func (s *SessionService) DuplicateSession(id string, newID string, newName string) error {
	// Get original session
	original, err := s.db.GetSession(id)
	if err != nil {
		return fmt.Errorf("failed to get original session: %w", err)
	}

	// Create duplicate
	duplicate := *original
	duplicate.ID = newID
	duplicate.Name = newName

	if err := s.db.CreateSession(&duplicate); err != nil {
		return fmt.Errorf("failed to create duplicate session: %w", err)
	}

	// Copy configs
	configs, err := s.db.GetSessionConfigs(id)
	if err != nil {
		return fmt.Errorf("failed to get session configs: %w", err)
	}

	for key, value := range configs {
		// Assume string type for now; in production, store type in DB
		if err := s.db.SetSessionConfig(newID, key, value, "string"); err != nil {
			return fmt.Errorf("failed to copy config: %w", err)
		}
	}

	return nil
}

// MoveSession moves a session to a new parent and position
func (s *SessionService) MoveSession(sessionID string, newParentID *string, newPosition int) error {
	return s.db.MoveSession(sessionID, newParentID, newPosition)
}

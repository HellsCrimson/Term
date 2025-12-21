package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/crypto/ssh"
)

type SftpService struct {
	terminalService   *TerminalService
	uploadMgr         *UploadManager
	sftpSessionsCache map[string]*sftpClientAdapter
}

func NewSFTPService(app *application.App, ts *TerminalService) *SftpService {
	return &SftpService{
		terminalService:   ts,
		uploadMgr:         NewUploadManager(app),
		sftpSessionsCache: make(map[string]*sftpClientAdapter),
	}
}

type FileList struct {
	RemotePath string      `json:"remote_path"`
	Files      []FileEntry `json:"files"`
}

type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	IsDir   bool   `json:"isDir"`
	ModTime int64  `json:"modTime"`
}

func (s *SftpService) HandleSSHFSList(sessionID string, remotePath string) (FileList, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return FileList{
			RemotePath: remotePath,
		}, fmt.Errorf("session ID required")
	}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return FileList{
			RemotePath: remotePath,
		}, fmt.Errorf("ssh session not found")
	}

	var err error
	var sftpClient *sftpClientAdapter
	if s.sftpSessionsCache != nil && s.sftpSessionsCache[sessionID] != nil {
		sftpClient = s.sftpSessionsCache[sessionID]
	} else {
		sftpClient, err = sftpNewClient(session.SSHClient)
		if err != nil {
			return FileList{
				RemotePath: remotePath,
			}, fmt.Errorf("failed to create sftp client: %v", err)
		}
		s.sftpSessionsCache[sessionID] = sftpClient
	}

	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		// Try resolve current directory to absolute path
		if p, err := sftpClient.RealPath("."); err == nil {
			remotePath = p
		} else {
			remotePath = "/"
		}
	}

	res := FileList{
		RemotePath: remotePath,
		Files:      make([]FileEntry, 0),
	}

	// Read directory
	entries, err := sftpClient.ReadDir(remotePath)
	if err != nil {
		return res, fmt.Errorf("failed to read directory: %v", err)
	}

	for _, fi := range entries {
		// Build child path using POSIX join
		p := posixJoin(remotePath, fi.Name())
		res.Files = append(res.Files, FileEntry{
			Name:    fi.Name(),
			Path:    p,
			Size:    fi.Size(),
			Mode:    fi.Mode().String(),
			IsDir:   fi.IsDir(),
			ModTime: fi.ModTime().Unix(),
		})
	}

	return res, nil
}

func (s *SftpService) HandleSSHFSDownload(sessionID string, remotePath string, dest string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session ID required")
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return fmt.Errorf("path query param required")
	}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return fmt.Errorf("ssh session not found")
	}

	var err error
	var sftpClient *sftpClientAdapter
	if s.sftpSessionsCache != nil && s.sftpSessionsCache[sessionID] != nil {
		sftpClient = s.sftpSessionsCache[sessionID]
	} else {
		sftpClient, err = sftpNewClient(session.SSHClient)
		if err != nil {
			return fmt.Errorf("failed to create sftp client: %v", err)
		}
		s.sftpSessionsCache[sessionID] = sftpClient
	}

	f, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %v", err)
	}
	defer f.Close()

	w, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer w.Close()

	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}

	return nil
}

func (s *SftpService) HandleSSHFSMkdir(sessionID string, remotePath string, recursive bool) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session ID required")
	}

	if remotePath == "" {
		return fmt.Errorf("path required")
	}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return fmt.Errorf("ssh session not found")
	}

	var err error
	var sftpClient *sftpClientAdapter
	if s.sftpSessionsCache != nil && s.sftpSessionsCache[sessionID] != nil {
		sftpClient = s.sftpSessionsCache[sessionID]
	} else {
		sftpClient, err = sftpNewClient(session.SSHClient)
		if err != nil {
			return fmt.Errorf("failed to create sftp client: %v", err)
		}
		s.sftpSessionsCache[sessionID] = sftpClient
	}

	var mkErr error
	if recursive {
		mkErr = sftpMkdirAll(sftpClient, remotePath)
	} else {
		mkErr = sftpClient.Mkdir(remotePath)
	}
	if mkErr != nil {
		return fmt.Errorf("failed to create directory: %v", mkErr)
	}

	return nil
}

func (s *SftpService) HandleSSHFSUpload(sessionID, localPath, destDir, jobID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID required")
	}

	if s.terminalService == nil {
		return fmt.Errorf("terminal service not available")
	}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return fmt.Errorf("ssh session not found")
	}

	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return fmt.Errorf("local path required")
	}

	// Validate local path exists and is a file
	lfi, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("local file not accessible: %v", err)
	}
	if lfi.IsDir() {
		return fmt.Errorf("local path is a directory")
	}

	// Destination dir (remote)
	destDir = strings.TrimSpace(destDir)
	if destDir == "" {
		destDir = "/"
	}

	// Use local filename as remote filename
	remotePath := posixJoin(destDir, fileBase(localPath))

	var sftpClient *sftpClientAdapter
	if s.sftpSessionsCache != nil && s.sftpSessionsCache[sessionID] != nil {
		sftpClient = s.sftpSessionsCache[sessionID]
	} else {
		sftpClient, err = sftpNewClient(session.SSHClient)
		if err != nil {
			return fmt.Errorf("failed to create sftp client: %v", err)
		}
		s.sftpSessionsCache[sessionID] = sftpClient
	}

	// Ensure directory exists (best-effort)
	_ = sftpMkdirAll(sftpClient, destDir)

	// Open local file for reading
	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %v", err)
	}
	defer src.Close()

	// Create remote destination file and copy
	dst, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %v", err)
	}
	defer dst.Close()

	// Progress-enabled copy
	if jobID != "" && s.uploadMgr != nil {
		// Publish initial state
		s.uploadMgr.Publish(jobID, UploadProgress{Total: lfi.Size(), Transferred: 0, Done: false, Error: ""})
		pr := &progressReader{r: src, total: lfi.Size(), jobID: jobID, mgr: s.uploadMgr}
		if _, err := io.Copy(dst, pr); err != nil {
			s.uploadMgr.Publish(jobID, UploadProgress{Total: lfi.Size(), Transferred: pr.transferred, Done: true, Error: err.Error()})
			return fmt.Errorf("failed to upload file: %v", err)
		}
		s.uploadMgr.Publish(jobID, UploadProgress{Total: lfi.Size(), Transferred: lfi.Size(), Done: true, Error: ""})
	} else {
		if _, err := io.Copy(dst, src); err != nil {
			return fmt.Errorf("failed to upload file: %v", err)
		}
	}

	return nil
}

func (s *SftpService) HandleSSHFSRename(sessionID, oldPath, newPath string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session ID required")
	}

	if oldPath == "" || newPath == "" {
		return fmt.Errorf("oldPath and newPath required")
	}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return fmt.Errorf("ssh session not found")
	}

	var err error
	var sftpClient *sftpClientAdapter
	if s.sftpSessionsCache != nil && s.sftpSessionsCache[sessionID] != nil {
		sftpClient = s.sftpSessionsCache[sessionID]
	} else {
		sftpClient, err = sftpNewClient(session.SSHClient)
		if err != nil {
			return fmt.Errorf("failed to create sftp client: %v", err)
		}
		s.sftpSessionsCache[sessionID] = sftpClient
	}

	if err := sftpClient.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename/move: %v", err)
	}

	return nil
}

func (s *SftpService) HandleSSHFSDelete(sessionID string, path string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session ID required")
	}

	if path == "" {
		return fmt.Errorf("path required")
	}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return fmt.Errorf("ssh session not found")
	}

	var err error
	var sftpClient *sftpClientAdapter
	if s.sftpSessionsCache != nil && s.sftpSessionsCache[sessionID] != nil {
		sftpClient = s.sftpSessionsCache[sessionID]
	} else {
		sftpClient, err = sftpNewClient(session.SSHClient)
		if err != nil {
			return fmt.Errorf("failed to create sftp client: %v", err)
		}
		s.sftpSessionsCache[sessionID] = sftpClient
	}

	if err := sftpRemoveAll(sftpClient, path); err != nil {
		return fmt.Errorf("failed to delete: %v", err)
	}
	return nil
}

func (s *SftpService) HandleSSHFSDownloadDir(sessionID string, remotePath string, localPath string) error {
	sessionID = strings.TrimSpace(sessionID)
	remotePath = strings.TrimSpace(remotePath)
	if sessionID == "" || remotePath == "" {
		return fmt.Errorf("sessionId and path required")
	}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return fmt.Errorf("ssh session not found")
	}

	var err error
	var sftpClient *sftpClientAdapter
	if s.sftpSessionsCache != nil && s.sftpSessionsCache[sessionID] != nil {
		sftpClient = s.sftpSessionsCache[sessionID]
	} else {
		sftpClient, err = sftpNewClient(session.SSHClient)
		if err != nil {
			return fmt.Errorf("failed to create sftp client: %v", err)
		}
		s.sftpSessionsCache[sessionID] = sftpClient
	}

	base := fileBase(remotePath)
	if base == "/" || base == "." || base == "" {
		base = "archive"
	}

	zipFileName := fmt.Sprintf("%s.zip", base)
	if localPath != "" {
		localPath = strings.TrimSpace(localPath)
		zipFileName = localPath
	}

	w, err := os.Create(zipFileName)
	if err != nil {
		return fmt.Errorf("failed to create local zip file: %v", err)
	}
	defer w.Close()

	if err := sftpZipDirToWriter(sftpClient, remotePath, w); err != nil {
		return fmt.Errorf("failed to zip directory: %v", err)
	}

	return nil
}

func (s *SftpService) HandleSSHFSSaveDir(sessionID string, remotePath string, dest string) error {
	sessionID = strings.TrimSpace(sessionID)

	if sessionID == "" || remotePath == "" || dest == "" {
		return fmt.Errorf("sessionId, path, dest required")
	}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return fmt.Errorf("ssh session not found")
	}

	var err error
	var sftpClient *sftpClientAdapter
	if s.sftpSessionsCache != nil && s.sftpSessionsCache[sessionID] != nil {
		sftpClient = s.sftpSessionsCache[sessionID]
	} else {
		sftpClient, err = sftpNewClient(session.SSHClient)
		if err != nil {
			return fmt.Errorf("failed to create sftp client: %v", err)
		}
		s.sftpSessionsCache[sessionID] = sftpClient
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer f.Close()

	if err := sftpZipDirToWriter(sftpClient, remotePath, f); err != nil {
		return fmt.Errorf("failed to zip directory: %v", err)
	}

	return nil
}

func sftpRemoveAll(c *sftpClientAdapter, p string) error {
	fi, err := c.Stat(p)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return c.Remove(p)
	}

	entries, err := c.ReadDir(p)
	if err != nil {
		return err
	}
	for _, e := range entries {
		child := posixJoin(p, e.Name())
		if e.IsDir() {
			if err := sftpRemoveAll(c, child); err != nil {
				return err
			}
		} else {
			if err := c.Remove(child); err != nil {
				return err
			}
		}
	}

	return c.RemoveDirectory(p)
}

func sftpZipDirToWriter(c *sftpClientAdapter, root string, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Ensure root is a directory
	fi, err := c.Stat(root)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("not a directory: %s", root)
	}

	// Walk recursively
	var walk func(dir, rel string) error
	walk = func(dir, rel string) error {
		entries, err := c.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			name := e.Name()
			abs := posixJoin(dir, name)
			relPath := path.Join(rel, name)
			if e.IsDir() {
				// Create dir header
				if !strings.HasSuffix(relPath, "/") {
					relPath += "/"
				}
				if _, err := zw.CreateHeader(&zip.FileHeader{Name: relPath, Method: zip.Deflate}); err != nil {
					return err
				}
				if err := walk(abs, relPath); err != nil {
					return err
				}
			} else {
				// Create file entry
				hdr := &zip.FileHeader{
					Name:   relPath,
					Method: zip.Deflate,
				}
				// Set permissions if available
				hdr.SetMode(e.Mode() & fs.ModePerm)
				fw, err := zw.CreateHeader(hdr)
				if err != nil {
					return err
				}
				rc, err := c.Open(abs)
				if err != nil {
					return err
				}
				if _, err := io.Copy(fw, rc); err != nil {
					rc.Close()
					return err
				}
				rc.Close()
			}
		}
		return nil
	}

	// Root folder name in the zip
	base := fileBase(root)
	if base == "/" || base == "." || base == "" {
		base = "archive"
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	if _, err := zw.CreateHeader(&zip.FileHeader{Name: base, Method: zip.Deflate}); err != nil {
		return err
	}
	return walk(root, base)
}

func (s *SftpService) HandleSSHFSSave(sessionID string, remotePath string, destPath string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session ID required")
	}

	if remotePath == "" || destPath == "" {
		return fmt.Errorf("'path' and 'dest' are required")
	}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return fmt.Errorf("ssh session not found")
	}

	var err error
	var sftpClient *sftpClientAdapter
	if s.sftpSessionsCache != nil && s.sftpSessionsCache[sessionID] != nil {
		sftpClient = s.sftpSessionsCache[sessionID]
	} else {
		sftpClient, err = sftpNewClient(session.SSHClient)
		if err != nil {
			return fmt.Errorf("failed to create sftp client: %v", err)
		}
		s.sftpSessionsCache[sessionID] = sftpClient
	}

	src, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %v", err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	return nil
}

func sftpNewClient(client *ssh.Client) (*sftpClientAdapter, error) {
	return newSFTPClientAdapter(client)
}

func (s *SftpService) ServiceShutdown() error {
	if s.sftpSessionsCache != nil {
		for _, c := range s.sftpSessionsCache {
			_ = c.Close()
		}
	}
	return nil
}

package main

import (
	"io"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// sftpClientAdapter wraps github.com/pkg/sftp.Client to keep httpserver decoupled
type sftpClientAdapter struct {
	c *sftp.Client
}

func newSFTPClientAdapter(client *ssh.Client) (*sftpClientAdapter, error) {
	c, err := sftp.NewClient(client, sftp.UseConcurrentWrites(true))
	if err != nil {
		return nil, err
	}
	return &sftpClientAdapter{c: c}, nil
}

func (a *sftpClientAdapter) Close() error { return a.c.Close() }

func (a *sftpClientAdapter) RealPath(p string) (string, error) { return a.c.RealPath(p) }

func (a *sftpClientAdapter) ReadDir(p string) ([]os.FileInfo, error) { return a.c.ReadDir(p) }

func (a *sftpClientAdapter) Open(p string) (io.ReadCloser, error) { return a.c.Open(p) }

func (a *sftpClientAdapter) Create(p string) (io.WriteCloser, error) { return a.c.Create(p) }

func sftpMkdirAll(a *sftpClientAdapter, p string) error { return a.c.MkdirAll(p) }

// Additional helpers
func (a *sftpClientAdapter) Stat(p string) (os.FileInfo, error) { return a.c.Stat(p) }
func (a *sftpClientAdapter) Mkdir(p string) error               { return a.c.Mkdir(p) }
func (a *sftpClientAdapter) Remove(p string) error              { return a.c.Remove(p) }
func (a *sftpClientAdapter) RemoveDirectory(p string) error     { return a.c.RemoveDirectory(p) }
func (a *sftpClientAdapter) Rename(oldp, newp string) error     { return a.c.Rename(oldp, newp) }

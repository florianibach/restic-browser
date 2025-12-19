package main

import (
	"path/filepath"
)

type RepoConfig struct {
	ID           string
	Path         string
	PasswordFile string
	NoLock       bool
}

var root = "/repo"
var repos = map[string]RepoConfig{
	"SRV000": {ID: "SRV000", Path: filepath.Join(root, "SRV000"), PasswordFile: "/secrets/srv000.txt", NoLock: true},
	"SRV001": {ID: "SRV001", Path: filepath.Join(root, "SRV001"), PasswordFile: "/secrets/srv001.txt", NoLock: true},
	"SRV002": {ID: "SRV002", Path: filepath.Join(root, "SRV002"), PasswordFile: "/secrets/srv002.txt", NoLock: true},
	"SRV003": {ID: "SRV003", Path: filepath.Join(root, "SRV003"), PasswordFile: "/secrets/srv003.txt", NoLock: true},
	"SRV004": {ID: "SRV004", Path: filepath.Join(root, "SRV004"), PasswordFile: "/secrets/srv004.txt", NoLock: true},
	"SRV005": {ID: "SRV005", Path: filepath.Join(root, "SRV005"), PasswordFile: "/secrets/srv005.txt", NoLock: true},
}

func GetRepo(repoId string) (RepoConfig, bool) {
	repoConfg, ok := repos[repoId]
	return repoConfg, ok
}

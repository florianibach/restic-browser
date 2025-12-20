package main

import "os"

type RepoConfig struct {
	ID       string
	Path     string
	Password string
	NoLock   bool
}

var root = "/repo"
var repos = map[string]RepoConfig{
	"REPO":   {ID: "REPO", Path: "/repo", Password: "", NoLock: true},
	"SRV000": {ID: "SRV000", Path: "/repo/srv000", Password: os.Getenv("RESTIC_PASSWORD_SRV000"), NoLock: true},
	"SRV001": {ID: "SRV001", Path: "/repo/srv001", Password: os.Getenv("RESTIC_PASSWORD_SRV001"), NoLock: true},
	"SRV002": {ID: "SRV002", Path: "/repo/srv002", Password: os.Getenv("RESTIC_PASSWORD_SRV002"), NoLock: true},
	"SRV003": {ID: "SRV003", Path: "/repo/srv003", Password: os.Getenv("RESTIC_PASSWORD_SRV003"), NoLock: true},
	"SRV004": {ID: "SRV004", Path: "/repo/srv004", Password: os.Getenv("RESTIC_PASSWORD_SRV004"), NoLock: true},
	"SRV005": {ID: "SRV005", Path: "/repo/srv005", Password: os.Getenv("RESTIC_PASSWORD_SRV005"), NoLock: true},
}

func GetRepo(repoId string) (RepoConfig, bool) {
	repoConfg, ok := repos[repoId]
	return repoConfg, ok
}

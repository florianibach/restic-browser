package main

type RepoConfig struct {
	ID       string
	Path     string
	Password string
	NoLock   bool
}

var root = "/repo"
var repos = map[string]RepoConfig{
	"SRV000": {ID: "SRV000", Path: "/repo/SRV000", Password: "CHANGE_ME", NoLock: true},
	"SRV001": {ID: "SRV001", Path: "/repo/SRV001", Password: "CHANGE_ME", NoLock: true},
	"SRV002": {ID: "SRV002", Path: "/repo/docs/srv002", Password: "qOWkdtoO/z1C9M6KbnlGcO5MvYPvcSitP+AAL2nrBsrMbDVo", NoLock: true},
	"SRV003": {ID: "SRV003", Path: "/repo/SRV003", Password: "CHANGE_ME", NoLock: true},
	"SRV004": {ID: "SRV004", Path: "/repo/SRV004", Password: "CHANGE_ME", NoLock: true},
	"SRV005": {ID: "SRV005", Path: "/repo/SRV005", Password: "CHANGE_ME", NoLock: true},
}

func GetRepo(repoId string) (RepoConfig, bool) {
	repoConfg, ok := repos[repoId]
	return repoConfg, ok
}

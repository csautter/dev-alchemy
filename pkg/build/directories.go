package build

import (
	"log"
	"os"
	"path/filepath"
)

type Directories struct {
	WorkingDir string
	ProjectDir string
	CacheDir   string
}

var (
	directoriesInstance *Directories
)

func GetDirectoriesInstance() *Directories {
	if directoriesInstance == nil {
		directoriesInstance = &Directories{}
		directoriesInstance.GetDirectories()
	}
	return directoriesInstance
}

func (u *Directories) GetDirectories() Directories {
	if u.WorkingDir == "" {
		if u.getWorkingDir() == "" {
			log.Fatalf("Working dir could not be determined.")
		}
	}
	if u.ProjectDir == "" {
		if u.determineTopLevelDirWithGit() == "" {
			log.Fatalf("Project dir could not be determined.")
		}
	}
	if u.CacheDir == "" {
		u.CacheDir = filepath.Join(u.ProjectDir, "cache")
	}
	return *u
}

func (u *Directories) determineTopLevelDirWithGit() string {
	dir := u.WorkingDir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Stat(gitPath)
		if err == nil && info.IsDir() {
			u.ProjectDir = dir
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func (u *Directories) getWorkingDir() string {
	if u.WorkingDir != "" {
		return u.WorkingDir
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	u.WorkingDir = dir
	return dir
}

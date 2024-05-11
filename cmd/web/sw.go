package main

import (
	"os"
	"path/filepath"
)

func (cfg *config) buildStaticFileList() error {
	err := filepath.Walk(cfg.serviceWorker.staticDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				cfg.serviceWorker.staticFileList = append(cfg.serviceWorker.staticFileList, path)
			}
			return nil
		})

	return err
}

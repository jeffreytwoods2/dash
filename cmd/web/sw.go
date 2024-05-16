package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func (cfg *config) buildStaticFileList() error {
	len_path_prefix := len(cfg.serviceWorker.staticDir)
	err := filepath.Walk(cfg.serviceWorker.staticDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				short_filepath := path[len_path_prefix:]
				// uri := fmt.Sprintf("https://dash.jwoods.dev/static%s", short_filepath)
				uri := fmt.Sprintf("http://localhost:9090/static%s", short_filepath)
				cfg.serviceWorker.staticFileList = append(cfg.serviceWorker.staticFileList, uri)
			}
			return nil
		})

	return err
}

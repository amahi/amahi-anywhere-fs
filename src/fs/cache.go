package main

import (
	"github.com/disintegration/imaging"
	"os"
	"path/filepath"
	"strings"
)

func thumbnailer(imagePath string, savePath string) error {
	//// source https://www.socketloop.com/tutorials/golang-generate-thumbnails-from-images
	img, err := imaging.Open(imagePath)
	if err != nil {
		return err
	}
	thumb := imaging.Thumbnail(img, 100, 100, imaging.CatmullRom)

	err = imaging.Save(thumb, savePath)
	if err != nil {
		return err
	}

	return nil
}

func fillCache(root string) error {
	filepath.Walk(root, getWalkFunction())
	return nil
}

func getWalkFunction() func(path string, info os.FileInfo, err error) error {
	return func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, ".fscache") {
			return nil
		}
		fi, _ := os.Stat(path)
		// if the current path is to directory, check if the .fscache/thumbnail subdirectories exists.
		if fi.IsDir() {
			// if the current path is directory, check for existence of .fscache and thumbnails
			thumbnailPath := filepath.Join(path, ".fscache/thumbnails")
			_, err := os.Stat(filepath.Join(thumbnailPath))
			if os.IsNotExist(err) {
				// if thumbnail directory doesn't exist, create it.
				os.MkdirAll(thumbnailPath, os.ModePerm)
			}
		} else {
			// if the path is to a file instead of directory
			parentDir := filepath.Dir(path)

			// this directory should exist because we
			// are walking down the tree and making
			// .fscache for each directory
			thumbnailDirPath := filepath.Join(parentDir, ".fscache/thumbnails")
			filename := filepath.Base(path)

			// TODO: check if the file represented by `path` is image or not
			thumbnailer(path, filepath.Join(thumbnailDirPath, filename))

		}
		return nil
	}
}

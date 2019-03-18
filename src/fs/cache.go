package main

import (
	"github.com/disintegration/imaging"
	"os"
	"path/filepath"
	"strings"
)

func thumbnailer(imagePath string, savePath string) error {
	//// source https://www.socketloop.com/tutorials/golang-generate-thumbnails-from-images
	log("path: ", imagePath)
	img, err := imaging.Open(imagePath)
	if err != nil {
		log("some error occurred")
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
			thumbnailDirPath := filepath.Join(path, ".fscache/thumbnails")
			_, err := os.Stat(filepath.Join(thumbnailDirPath))
			if os.IsNotExist(err) {
				// if thumbnail directory doesn't exist, create it.
				os.MkdirAll(thumbnailDirPath, os.ModePerm)
			}
		} else {
			// if the path is to a file instead of directory
			parentDir := filepath.Dir(path)
			filename := filepath.Base(path)

			// this directory should exist because we are walking
			// down the tree and making .fscache for each directory
			// EDIT: this is not necessary. Human intervention can change
			// the directories any time.
			thumbnailDirPath := filepath.Join(parentDir, ".fscache/thumbnails")
			_, err := os.Stat(thumbnailDirPath)
			if os.IsNotExist(err) {
				os.MkdirAll(thumbnailDirPath, os.ModePerm)
			}
			imageInfo, _ := os.Stat(path)
			thumbnailPath := filepath.Join(thumbnailDirPath, filename)
			thumbnailInfo, err := os.Stat(thumbnailPath)
			if os.IsNotExist(err) || imageInfo.ModTime().After(thumbnailInfo.ModTime()) {
				thumbnailer(path, thumbnailPath)
			}
		}
		return nil
	}
}

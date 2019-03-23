package main

import (
	"github.com/disintegration/imaging"
	"os"
	"path/filepath"
	"strings"
)

func thumbnailer(imagePath string, savePath string) error {
	img, err := imaging.Open(imagePath)
	log("path: ", imagePath)
	if err != nil {
		log("some error occurred for file: ", imagePath)
		return err
	}
	imgX := img.Bounds().Max.X
	imgY := img.Bounds().Max.Y

	thumbX := (imgX * 100 * 2) / (imgX + imgY)
	thumbY := (imgY * 100 * 2) / (imgX + imgY)

	thumb := imaging.Thumbnail(img, thumbX, thumbY, imaging.NearestNeighbor)

	os.MkdirAll(filepath.Dir(savePath), os.ModePerm)
	err = imaging.Save(thumb, savePath)
	if err != nil {
		return err
	}

	return nil
}

func fillCache(root string) error {
	filepath.Walk(root, walkFunc)
	return nil
}

func walkFunc(path string, info os.FileInfo, err error) error {
	if strings.Contains(path, ".fscache") {
		return nil
	}
	fi, _ := os.Stat(path)
	if ! fi.IsDir() {
		parentDir := filepath.Dir(path)
		filename := filepath.Base(path)

		thumbnailDirPath := filepath.Join(parentDir, ".fscache/thumbnails")
		thumbnailPath := filepath.Join(thumbnailDirPath, filename)
		imageInfo, _ := os.Stat(path)
		thumbnailInfo, err := os.Stat(thumbnailPath)
		if os.IsNotExist(err) || imageInfo.ModTime().After(thumbnailInfo.ModTime()) {
			thumbnailer(path, thumbnailPath)
		}
	}
	return nil
}

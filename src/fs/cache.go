package main

import (
	"encoding/json"
	"fmt"
	"github.com/dhowden/tag"
	"github.com/disintegration/imaging"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func thumbnailer(imagePath string, savePath string) error {
	img, err := imaging.Open(imagePath)
	if err != nil {
		log(`Error opening file at location: "%s" as image. Error is: "%s"`, imagePath, err.Error())
		return err
	}
	imgX := img.Bounds().Max.X
	imgY := img.Bounds().Max.Y

	thumbX := (imgX * 100 * 2) / (imgX + imgY)
	thumbY := (imgY * 100 * 2) / (imgX + imgY)

	thumb := imaging.Thumbnail(img, thumbX, thumbY, imaging.Box)

	os.MkdirAll(filepath.Dir(savePath), os.ModePerm)
	err = imaging.Save(thumb, savePath)
	if err != nil {
		log(`Error saving image thumbnail for file at location: "%s". Error is: "%s"`, imagePath, err.Error())
		return err
	}

	return nil
}

func cacheMetadataPicture(mediaPath string, savePathPrefix string) error {
	f, err := os.Open(mediaPath)
	if err != nil {
		log("error opening file: %s", mediaPath)
		return err
	}
	t, err := tag.ReadFrom(f)
	if err != nil {
		log(`Error fetching metadata for file: "%s". Error is: "%s"`, mediaPath, err.Error())
		return err
	}

	if pic := t.Picture(); pic != nil && pic.MIMEType != "" {
		fmt.Println("File path: ", mediaPath)
		json.NewEncoder(os.Stdout).Encode(pic)
		for k, v := range EncodingMap {
			if v == pic.MIMEType {
				savePath := savePathPrefix + k
				err = os.MkdirAll(filepath.Dir(savePath), os.ModePerm)
				if err != nil {
					log(`Error creating parent directory for file: "%s". Error is: "%s"`, savePath, err.Error())
					return err
				}
				err = ioutil.WriteFile(savePath, pic.Data, os.ModePerm)
				if err != nil {
					log(`Error saving image thumbnail for file at location: "%s". Error is: "%s"`, savePath, err.Error())
					return err
				}
				log("Thumbnail image saved for file: %s", mediaPath)
				return nil
			}
		}
	}
	log("Thumbnail image not found for file: %s", mediaPath)
	return nil
}

func fillCache(root string) error {
	filepath.Walk(root, fillCacheWalkFunc)
	return nil
}

func getThumbnailPath(path string) string {
	parentDir := filepath.Dir(path)
	filename := filepath.Base(path)
	thumbnailDirPath := filepath.Join(parentDir, ".fscache/thumbnails")
	thumbnailPath := filepath.Join(thumbnailDirPath, filename)
	_, err := os.Stat(thumbnailPath)
	if os.IsNotExist(err) {
		files, err := ioutil.ReadDir(thumbnailDirPath)
		if err == nil {
			for _, f := range files {
				fmt.Println(f.Name())
				if strings.Contains(f.Name(), filename) {
					thumbnailPath = filepath.Join(thumbnailDirPath, f.Name())
					break
				}
			}
		}
	}
	return thumbnailPath
}

func fillCacheWalkFunc(path string, info os.FileInfo, err error) error {
	if strings.Contains(path, ".fscache") {
		return nil
	}
	if ! info.IsDir() {
		thumbnailPath := getThumbnailPath(path)
		thumbnailInfo, err := os.Stat(thumbnailPath)
		if os.IsNotExist(err) || info.ModTime().After(thumbnailInfo.ModTime()) {
			contentType := getContentType(path)
			if strings.Contains(contentType, "image") {
				thumbnailer(path, thumbnailPath)
			} else {
				cacheMetadataPicture(path, thumbnailPath)
			}
		}
	} else {
		watcher.Add(path)
	}
	return nil
}

func removeCache(root string) error {
	filepath.Walk(root, removeCacheWalkFunc)
	return nil
}

func removeCacheWalkFunc(path string, info os.FileInfo, err error) error {
	if strings.Contains(path, ".fscache") {
		return nil
	}
	parentDir := filepath.Dir(path)
	filename := filepath.Base(path)

	thumbnailDirPath := filepath.Join(parentDir, ".fscache/thumbnails")
	thumbnailPath := filepath.Join(thumbnailDirPath, filename)
	_, err = os.Stat(thumbnailPath)
	if ! os.IsNotExist(err) {
		err := os.Remove(thumbnailPath)
		if err != nil {
			log(`Error while deleting cache file. Error: "%s"`, err.Error())
		}
	}
	watcher.Remove(path)
	return nil
}

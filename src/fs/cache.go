package main

import (
	"github.com/disintegration/imaging"
	"github.com/frolovo22/tag"
	"os"
	"path/filepath"
	"strings"
)

func thumbnailer(imagePath string, savePath string) error {
	img, err := imaging.Open(imagePath)
	if err != nil {
		//log(`Error opening file at location: "%s" as image. Error is: "%s"`, imagePath, err.Error())
		logging.Error(`Error opening file at location: "%s" as image. Error is: "%s"`, imagePath, err.Error())
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
		//log(`Error saving image thumbnail for file at location: "%s". Error is: "%s"`, imagePath, err.Error())
		logging.Error(`Error saving image thumbnail for file at location: "%s". Error is: "%s"`, imagePath, err.Error())
		return err
	}

	return nil
}

func cacheMetadataPicture(path string, savePath string) error {
	tags, err := tag.ReadFile(path)
	if err != nil {
		logging.Error("error opening file: %s", err.Error())
		return err
	}

	picture, err := tags.GetPicture()
	if err != nil {
		logging.Error(`Error getting media picture: %s`, err.Error())
		return err
	}

	if picture != nil {
		// song.mp3
		fileName := filepath.Base(path)
		index := strings.Index(fileName, ".")
		// song_artwork.jpg
		artworkName := fileName[:index] + "_artwork" + ".jpg"
		savePath = filepath.Join(filepath.Dir(savePath), artworkName)

		if err = os.MkdirAll(filepath.Dir(savePath), os.ModePerm); err != nil {
			logging.Error(`Error creating parent directory for file: "%s". Error is: "%s"`, savePath, err.Error())
			return err
		}

		if err = imaging.Save(picture, savePath); err != nil {
			logging.Error(`Error saving image thumbnail for file at location: "%s". Error is: "%s"`, savePath, err.Error())
			return err
		}
		logging.Info("Thumbnail image saved for file: %s", path)
		return nil
	}

	logging.Info("Thumbnail image not found for file: %s", path)
	return nil
}


func fillCache(root string) error {
	filepath.Walk(root, fillCacheWalkFunc)
	return nil
}

func fillCacheWalkFunc(path string, info os.FileInfo, err error) error {
	defer func() {
		if v := recover(); v != nil {
			logging.Fatal("Panic while creating thumbnail: %s", v)
		}
	}()

	if strings.Contains(path, ".fscache") {
		return nil
	}
	if ! info.IsDir() {
		parentDir := filepath.Dir(path)
		filename := filepath.Base(path)

		thumbnailDirPath := filepath.Join(parentDir, ".fscache/thumbnails")
		thumbnailPath := filepath.Join(thumbnailDirPath, filename)
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
			//log(`Error while deleting cache file. Error: "%s"`, err.Error())
			logging.Error(`Error while deleting cache file. Error: "%s"`, err.Error())
		}
	}
	err = watcher.Remove(path)
	if err != nil {
		//log(fmt.Sprintf("Error while removing file from watcher: %s", err))
		logging.Error("Error while removing file from watcher: %s", err)
	}
	return nil
}

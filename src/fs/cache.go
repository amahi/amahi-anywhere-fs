package main

import "github.com/disintegration/imaging"

func thumbnailer(image_path string, save_path string) error {
	//// source https://www.socketloop.com/tutorials/golang-generate-thumbnails-from-images
	img, err := imaging.Open(image_path)
	if err != nil {
		return err
	}
	thumb := imaging.Thumbnail(img, 100, 100, imaging.CatmullRom)

	err = imaging.Save(thumb, save_path)
	if err != nil {
		return err
	}
}

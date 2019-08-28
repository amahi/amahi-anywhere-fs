package main

import (
	"github.com/dhowden/tag"
	"os"
)

type Metadata struct {
	Format      string `json:"format"`
	FileType    string `json:"file_type"`
	Title       string `json:"title"`
	Album       string `json:"album"`
	Artist      string `json:"artist"`
	AlbumArtist string `json:"album_artist"`
	Composer    string `json:"composer"`
	Genre       string `json:"genre"`
	Year        int    `json:"year"`
	Lyrics      string `json:"lyrics"`
	Comment     string `json:"comment"`
}

func getMetadata(t tag.Metadata) (*Metadata) {
	m := &Metadata{
		Format:      string(t.Format()),
		FileType:    string(t.FileType()),
		Title:       string(t.Title()),
		Album:       string(t.Album()),
		Artist:      string(t.Artist()),
		AlbumArtist: string(t.AlbumArtist()),
		Composer:    string(t.Composer()),
		Genre:       string(t.Genre()),
		Year:        int(t.Year()),
		Lyrics:      string(t.Lyrics()),
		Comment:     string(t.Comment()),
	}
	return m
}

func getMetadataFromPath(path string) (*Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	t, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	return getMetadata(t), nil
}

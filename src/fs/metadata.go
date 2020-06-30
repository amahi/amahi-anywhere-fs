package main

import (
	"github.com/frolovo22/tag"
)

type Metadata struct {
	Tag         string      `json:"tag"`
	Title       string      `json:"title"`
	Album       string      `json:"album"`
	Artist      string      `json:"artist"`
	AlbumArtist string      `json:"album_artist"`
	Composer    string      `json:"composer"`
	Genre       string      `json:"genre"`
	Year        int         `json:"year"`
	TrackNumber int         `json:"track_number"`
	AlbumArtwork string     `json:"album_artwork"`
}

func getMetaData(tags tag.Metadata) (*Metadata) {

	title, _ := tags.GetTitle()
	album, _ := tags.GetAlbum()
	tagVersion := tags.GetVersion()
	artist, _ := tags.GetArtist()
	albumArtist, _ := tags.GetAlbumArtist()
	genre, _ := tags.GetGenre()
	trackNumber, _, _ := tags.GetTrackNumber()
	composer, _ := tags.GetComposer()
	year, _ := tags.GetYear()

	metadata := &Metadata{
		Tag:         tagVersion.String(),
		Title:       title,
		Album:       album,
		Artist:      artist,
		AlbumArtist: albumArtist,
		Composer:    composer,
		Genre:       genre,
		Year:        year,
		TrackNumber: trackNumber,
	}
	return metadata
}

func getMetadataByPath(path string) (*Metadata, error) {
	tags, err := tag.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return getMetaData(tags), nil
}
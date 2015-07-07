// movie.go
package movie

import (
	"torrent"
)

type Movie struct {
	ImdbId         string
	FilmAffinityId string
	Title          string
	OriginalTitle  string
	Year           string
	Genre          string
	Rating         float32
	Image          string
	Description    string
	Url            string
	Torrents       map[string]*torrent.Torrent
}

func (m *Movie) AddTorrent(key string, t *torrent.Torrent) {
	if m.Torrents == nil {
		m.Torrents = make(map[string]*torrent.Torrent)
	}
	m.Torrents[key] = t
}

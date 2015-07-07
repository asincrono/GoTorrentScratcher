// torrent.go
package torrent

type Torrent struct {
	Quality  string // 720p
	URL      string
	Magnet   string
	Size     int
	Filesize string // 812.24 MB
	Seeds    uint16
	Peers    uint16
}

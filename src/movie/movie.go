// movie.go
package movie

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"torrent"

	"regexp"

	"github.com/PuerkitoBio/goquery"
)

const (
	UrlSep                 = "/"
	FilmAffinityURL        = "http://www.filmaffinity.com"
	FilmAffinitySearch     = "es/search.php?stext=%s&stype=title"
	FilmAffinityOderByYear = "oderby=year"

	EliteTorrentURL = "http://www.elitetorrent.net"
	CategoriaHDRIP  = "categoria/13/peliculas-hdrip"
	ModeList        = "modo:listado"
	OrderScore      = "orden:valoracion"
	Page            = "pag:"

	OMDBApiUrl   = "http://www.omdbapi.com"
	OMDBApiQuery = "?t=%s&type=movie"

	IMDBUrl           = "http://www.imdb.com"
	IMDBQuery         = "find?q=%s&s=all"
	IMDBAdvancedQuery = "search/title?production_status=released&sort=year,desc&title=%s&title_type=feature&view=simple"
)

type Movie struct {
	ImdbId         string
	FilmAffinityId string

	Title         string
	OriginalTitle string

	Year    string
	Relased string
	Country string

	Genre string
	Rated string

	Duration string

	Rating     string
	Metascore  string
	ImdbRating string
	ImdbVotes  string

	Director string
	Writer   string
	Actors   string
	Plot     string

	Image       string
	Description string
	Url         string
	Web         string
	ImdbUrl     string
	FileSize    string
	Torrents    map[string]*torrent.Torrent

	updated bool
}

func (m *Movie) GetFileSize() float32 {
	val, _ := strconv.ParseFloat(m.FileSize, 32)
	return float32(val)
}

func (m *Movie) IsUpdated() bool {
	return m.updated
}

func (m *Movie) setUpdated() {
	m.updated = true
}

func (m *Movie) GetRating() float32 {
	val, _ := strconv.ParseFloat(m.Rating, 32)
	return float32(val)
}

func (m *Movie) GetMetascore() (rating int) {
	val, _ := strconv.ParseInt(m.Metascore, 10, 0)
	return int(val)
}

func (m *Movie) GetImdbRating() (rating float32) {
	val, _ := strconv.ParseFloat(m.ImdbRating, 32)
	return float32(val)
}

func (m *Movie) GetImdbVotes() (rating uint8, err error) {
	val, err := strconv.ParseUint(m.ImdbVotes, 10, 8)
	rating = uint8(val)
	return
}

func (m *Movie) AddTorrent(key string, t *torrent.Torrent) {
	if m.Torrents == nil {
		m.Torrents = make(map[string]*torrent.Torrent)
	}
	m.Torrents[key] = t
}

func CleanTitle(title string) string {
	byteStr := []byte(title)
	parenthesesRegex, _ := regexp.Compile("\\(.*?\\)")
	bracketsRegex, _ := regexp.Compile("\\[.*?\\]")

	byteStr = parenthesesRegex.ReplaceAll(byteStr, []byte(""))
	byteStr = bracketsRegex.ReplaceAll(byteStr, []byte(""))

	cTitle := string(byteStr)

	cTitle = strings.TrimSpace(cTitle)
	return cTitle
}

//Devuelve verdadero si "title", después de eliminar cualquier caracter distinto de los alfanuméricos
//del castellano, descompuesto en palabras, es un subconjunto de "inTitle". Falso en caso contrario.
func TitleMatch(title string, inTitle string) bool {
	titleByte := []byte(title)
	inTitleByte := []byte(inTitle)

	notAllowed, _ := regexp.Compile("[^a-zA-Z0-1 áéíóúñ]+")
	spaces, _ := regexp.Compile("\\s\\s+")
	titleByte = notAllowed.ReplaceAll(titleByte, []byte(""))
	titleByte = spaces.ReplaceAll(titleByte, []byte(" "))

	titleByte = bytes.ToLower(titleByte)
	inTitleByte = bytes.ToLower(inTitleByte)

	titleWordsByte := bytes.Split(titleByte, []byte(" "))

	for _, wordByte := range titleWordsByte {
		if !bytes.Contains(inTitleByte, wordByte) {
			return false
		}
	}

	return true
}

func (m *Movie) EnrichWithOmdbApi() {
	var title string
	if m.OriginalTitle != "" {
		title = m.OriginalTitle
	} else {
		title = m.Title
	}

	query := fmt.Sprintf(OMDBApiQuery, title)

	url := strings.Join([]string{OMDBApiUrl, query}, UrlSep)
	if res, err := http.Get(url); err != nil {
		log.Fatal(err)
	} else {
		defer res.Body.Close()

		rawJson, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		if err := json.Unmarshal(rawJson, m); err != nil {
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				m.updated = true
				// Parece que el títlo tiene espacios al final.
				m.Title = strings.TrimSpace(m.Title)
			} else {
				m.updated = false
			}
		} else {
			m.updated = false
		}
	}
}

func (m *Movie) GetMovieFromPath(path string) {
	url := EliteTorrentURL + path
	log.Println("Retrieving", path+".")

	if res, err := http.Get(url); err != nil {
		log.Fatal(err)
	} else {

		doc, err := goquery.NewDocumentFromResponse(res)

		if err != nil {
			log.Fatal(err)
		}

		m.Url = url

		title := doc.Find("#box-ficha > h2").Text()

		m.Title = CleanTitle(title)

		m.Description = doc.Find("p.descrip").Eq(1).Text()
		m.Rating = doc.Find("span.valoracion").Text()

		imgName, _ := doc.Find("img.imagen_ficha").Attr("src")

		m.Image = EliteTorrentURL + imgName

		var torrent torrent.Torrent

		torrent.Magnet, _ = doc.Find("a[href^=magnet]").Attr("href")
		torrent.Filesize = doc.Find("dl.info-tecnica dd").Eq(3).Text()

		seedsClientsText := doc.Find("div.ppal").Text()

		seedsClientsArr := strings.Fields(seedsClientsText)

		seeds, _ := strconv.Atoi(seedsClientsArr[1])
		peers, _ := strconv.Atoi(seedsClientsArr[4])

		torrent.Seeds = uint16(seeds)
		torrent.Peers = uint16(peers)

		m.AddTorrent("720p", &torrent)
	}
}

func (m *Movie) EnrichWithFilmAffinity(overwrite bool) {
	var title string

	if m.OriginalTitle != "" {
		title = m.OriginalTitle
	} else {
		title = m.Title
	}

	auxTitle := strings.Join(strings.Split(title, " "), "+")
	fmt.Printf("(EnrichWithFilmAffinity) auxTitle: \"%s\".\n", auxTitle)

	query := fmt.Sprintf(FilmAffinitySearch, auxTitle)
	url := strings.Join([]string{FilmAffinityURL, query}, UrlSep)
	fmt.Println("(EnrichWithFilmAffinity) url:", url)

	if doc, err := goquery.NewDocument(url); err != nil {
		log.Fatal(err)
	} else {
		// Two posibilities: We get the movie page or we get a list of movies.

		selection := doc.Find("[property='og:title']")
		// Property og:title implies that we have only one match -> We are at the movie page.

		found := false
		if selection.Length() == 0 {
			// We are not at the title page, we have more than one result.
			selection = doc.Find(".item-search .mc-title a")

			selection.EachWithBreak(func(i int, s *goquery.Selection) bool {
				newTitle := CleanTitle(s.First().Text())

				fmt.Printf("(EnrichWithFilmAffinity) Comparando \"%s\" con \"%s\".\n", newTitle, title)
				// Once we find the right title we get out.
				if TitleMatch(newTitle, title) {
					fmt.Printf("(EnrichWithFilmAffinity) Match Found: \"%s\" - \"%s\"\n", newTitle, title)
					found = true

					// Here we stablish the right movie selection
					selection = s
					return false
				} else {
					return true
				}
			})

			movieUrl, ok := selection.Attr("href")
			if !ok {
				log.Println("No se encuentra la url de la peli.")
				return
			}
			url = strings.Join([]string{FilmAffinityURL, movieUrl}, UrlSep)
			// Now we get the right movie page.
			doc, err = goquery.NewDocument(url)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			// One result.
			found = true
		}

		if !found {
			log.Println("Movie not found!")
			return
		}

		selection = doc.Find("dl.movie-info").First().Children()
		if selection.Length() == 0 {
			log.Println("No encuentra nada para dl.movie_info")
		}

		selection.Each(func(i int, s *goquery.Selection) {

			switch s.First().Text() {
			case "Año":
				if overwrite || m.Year == "" {
					s = s.Next()
					m.Year = s.First().Text()
				}

			case "Duración":
				if overwrite || m.Duration == "" {
					s = s.Next()
					m.Duration = s.First().Text()
				}

			case "País":
				if overwrite || m.Country == "" {
					s = s.Next()
					m.Country = s.First().Text()
				}
			case "Director":
				if overwrite || m.Director == "" {
					s = s.Next()
					m.Director = s.First().Text()
				}
			case "Guión":
				if overwrite || m.Writer == "" {
					s = s.Next()
					m.Writer = s.First().Text()
				}
			case "Género":
				if overwrite || m.Genre == "" {
					s = s.Next()

					s.Find("a").Each(func(i int, s *goquery.Selection) {
						m.Genre = strings.Join([]string{m.Genre, s.First().Text()}, "|")
					})
				}
			case "Sinopsis":
				if overwrite || m.Plot == "" {
					s = s.Next()
					m.Plot = strings.TrimRight(s.First().Text(), " \n")
				}
			case "Web oficial":
				if overwrite || m.Web == "" {
					s = s.Next()
					m.Web, _ = s.Attr("href")
				}
			}
		})

		if overwrite || m.FilmAffinityId == "" {
			m.FilmAffinityId, _ = doc.Find("div.rate-movie-box").Attr("data-movie-id")
		}

		if overwrite || m.OriginalTitle == "" {
			title := doc.Find("dd").Eq(0).Text()
			m.OriginalTitle = CleanTitle(title)
		}
	}
}

func (m *Movie) EnrichWithImdbSimpleSearch(overwrite bool) bool {
	var title string

	if m.OriginalTitle != "" {
		title = m.OriginalTitle
	} else {
		title = m.Title
	}

	query := fmt.Sprintf(IMDBQuery, title)
	url := strings.Join([]string{IMDBUrl, query}, "/")

	log.Println("(EnrichWithImdbSearch) url:", url)

	// Dos posibilidades: Página directa de la película o lista de resultados.

	if doc, err := goquery.NewDocument(url); err != nil {
		log.Fatal(err)
	} else {

		// Aquí hay que discriminar si es la página de la película o...
		// una lista de resutlados.
		selection := doc.Find(".title").Find("a")

		if selection.Length() == 0 {
			log.Println("Movie not found: " + m.Title)
			return false
		} else {
			// Iteramos selection hasta encontar el primer resultado de peli (su href contiene /title/)??
			var link string
			var newTitle string

			selection.EachWithBreak(func(i int, s *goquery.Selection) bool {
				newTitle = s.First().Text()
				if TitleMatch(newTitle, title) {
					link, _ = s.Attr("href")
					return false
				}
				return true
			})

			if overwrite || m.ImdbUrl == "" {
				m.ImdbUrl = IMDBUrl + link
			}

			if overwrite || m.ImdbId == "" {
				m.ImdbId = link[len("/title/") : len(link)-1]
			}
		}

		if doc, err = goquery.NewDocument(m.ImdbUrl); err != nil {
			log.Fatal(err)
		} else {
			if overwrite || m.Genre == "" {
				m.Genre = doc.Find("[itemprop=genre]").Eq(0).Text()
			}
			if overwrite || m.ImdbRating == "" {
				m.ImdbRating = doc.Find("[itemprop=aggregateRating]").Find("[itemprop=ratingValue]").Text()
			}
			if overwrite || m.Director == "" {
				m.Director = doc.Find("[itemprop=director]").Find("[itemprop=name]").Text()
			}
			if overwrite || m.Duration == "" {
				m.Duration = doc.Find("[itemprop=duration]").Last().Text()
			}
		}
	}
	return true
}

func (m *Movie) EnrichWithImdbAdvancedSearch(overwrite bool) bool {
	var title string

	if m.OriginalTitle != "" {
		title = m.OriginalTitle
	} else {
		title = m.Title
	}

	query := fmt.Sprintf(IMDBAdvancedQuery, title)
	url := strings.Join([]string{IMDBUrl, query}, "/")

	log.Println("(EnrichWithImdbSearch) url:", url)

	// Dos posibilidades: Página directa de la película o lista de resultados.

	if doc, err := goquery.NewDocument(url); err != nil {
		log.Fatal(err)
	} else {

		// Aquí hay que discriminar si es la página de la película o...
		// una lista de resutlados.
		selection := doc.Find(".title").Find("a")

		if selection.Length() == 0 {
			log.Println("Movie not found: " + m.Title)
			return false
		} else {
			// Iteramos selection hasta encontar el primer resultado de peli (su href contiene /title/)??
			var link string
			var newTitle string

			selection.EachWithBreak(func(i int, s *goquery.Selection) bool {
				newTitle = s.First().Text()
				if TitleMatch(newTitle, title) {
					link, _ = s.Attr("href")
					return false
				}
				return true
			})

			if overwrite || m.ImdbUrl == "" {
				m.ImdbUrl = IMDBUrl + link
			}

			if overwrite || m.ImdbId == "" {
				m.ImdbId = link[len("/title/") : len(link)-1]
			}
		}

		if doc, err = goquery.NewDocument(m.ImdbUrl); err != nil {
			log.Fatal(err)
		} else {
			if overwrite || m.Genre == "" {
				m.Genre = doc.Find("[itemprop=genre]").Eq(0).Text()
			}
			if overwrite || m.ImdbRating == "" {
				m.ImdbRating = doc.Find("[itemprop=aggregateRating]").Find("[itemprop=ratingValue]").Text()
			}
			if overwrite || m.Director == "" {
				m.Director = doc.Find("[itemprop=director]").Find("[itemprop=name]").Text()
			}
			if overwrite || m.Duration == "" {
				m.Duration = doc.Find("[itemprop=duration]").Last().Text()
			}
		}
	}
	return true
}

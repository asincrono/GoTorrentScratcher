// movie.go
package movie

import (
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
	FilmAffinitySearch     = "es/search.php?stext=%s&stype=all"
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

func (m *Movie) Get() uint8 {
	val, _ := strconv.ParseUint(m.ImdbVotes, 10, 8)
	return uint8(val)
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
			} else {
				m.updated = false
			}
		} else {
			m.updated = false
		}
	}
}

func removeParenthesesAndBracketsContent(s string) (out string) {
	modStr := []byte(s)
	parenthesesExpr, err := regexp.Compile("\\(.*?\\)")
	if err != nil {
		log.Fatal(err)
	}

	bracketsExpr, err := regexp.Compile("\\[.*?\\]")
	if err != nil {
		log.Fatal(err)
	}

	modStr = parenthesesExpr.ReplaceAll(modStr, []byte(""))
	modStr = bracketsExpr.ReplaceAll(modStr, []byte(""))
	out = string(modStr)
	return
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

		m.Title = removeParenthesesAndBracketsContent(title)
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

func (m *Movie) EnrichWithFilmAffinity() {
	var title string

	if m.OriginalTitle != "" {
		title = m.OriginalTitle
	} else {
		title = m.Title
	}

	query := fmt.Sprintf(FilmAffinitySearch, title)
	url := strings.Join([]string{FilmAffinityURL, query}, UrlSep)

	if doc, err := goquery.NewDocument(url); err != nil {
		log.Fatal(err)
	} else {
		selection := doc.Find("[property='og:title']").Find(".item-search").Find(".mc-title").Find(".mc-title a")
		// Property og:title imply that we have only one match -> We are at the movie page.
		if selection.Length() == 0 {
			// Many results
			selection = doc.Find(".item-search .mc-title a")
			if selection.Length() == 0 {
				log.Fatal("No encuentra nada.")
			}
			selection.EachWithBreak(func(i int, s *goquery.Selection) bool {
				newTitle := cleanTitle(s.First().Text())

				fmt.Println("Title:", newTitle)
				if title == newTitle {
					return false
				} else {
					return true
				}
			})
			// Once we find the right title.

		} else {
			// One result.
		}
		movieUrl, ok := selection.Attr("href")
		if !ok {
			log.Fatal("No se encuentra la url de la peli.")
		}
		url = strings.Join([]string{FilmAffinityURL, movieUrl}, UrlSep)
		doc, err := goquery.NewDocument(url)
		if err != nil {
			log.Fatal(err)
		}

		m.FilmAffinityId, _ = doc.Find("div.rate-movie-box").Attr("data-movie-id")
		title := doc.Find("dd").Eq(0).Text()

		m.OriginalTitle = cleanTitle(title)
	}
}

func cleanTitle(title string) string {

	cleanTitleByte := []byte(title)
	parenthesesExpr, _ := regexp.Compile("\\(.*?\\)")
	bracketsExpr, _ := regexp.Compile("\\[.*?\\]")

	cleanTitleByte = parenthesesExpr.ReplaceAll(cleanTitleByte, []byte(""))
	cleanTitleByte = bracketsExpr.ReplaceAll(cleanTitleByte, []byte(""))

	cleanTitleStr := string(cleanTitleByte)

	cleanTitleStr = strings.TrimFunc(cleanTitleStr, func(r rune) bool {
		if r == ' ' || r == '\n' || r == '\t' {
			return true
		} else {
			return false
		}
	})
	return cleanTitleStr
}

//public Movie enrichMovieWithFilmAffinity(Movie movie) {
//        try {
//            String url = "http://www.filmaffinity.com/es/search.php?stext={title}&stype=all";
//            String title = URLEncoder.encode(movie.getTitle(), "UTF8");
//            url = url.replace("{title}", title);
//            Document doc = Jsoup.connect(url).get();
//            if (doc.select("[property=og:title]").size() == 0) {
//                // several results found, take the first
//                Element firstResult = doc.select(".item-search .mc-title a").first();
//                if (firstResult == null) {
//                    // filmaffinity search engine failed
//                    log.warn("FilmAffinity 404: " + movie.getTitle());
//                    return movie;
//                }
//                url = "http://www.filmaffinity.com" + firstResult.attr("href");
//                doc = Jsoup.connect(url).get();
//            }
//            movie.setFilmAffinityId(doc.select("div.rate-movie-box").attr("data-movie-id"));
//            Elements movieInfo = doc.select("dl.movie-info");
//            String originalTitle = movieInfo.select("dd").eq(0).text();
//            originalTitle = originalTitle
//                    .replaceAll("\\([^\\(]*\\)", "")
//                    .replaceAll("\\[[^\\(]*\\]", "")
//                    .replaceAll("aka$", "")
//                    .trim();
//            movie.setOriginalTitle(originalTitle);
//            movie.setDescription(movieInfo.select("dd").eq(11).text());
//        } catch (IOException ex) {
//            log.warn(ex.getMessage());
//        }
//        return movie;
//    }

func (m *Movie) EnrichWithImdbSearch() {
	var title string

	if m.OriginalTitle != "" {
		title = m.OriginalTitle
	} else {
		title = m.Title
	}

	query := fmt.Sprintf(IMDBAdvancedQuery, title)
	url := strings.Join([]string{IMDBUrl, query}, "/")

	if doc, err := goquery.NewDocument(url); err != nil {
		log.Fatal(err)
	} else {
		selection := doc.Find(".title").Find("a")

		if selection.Length() == 0 {
			log.Fatal("IMDB search 404: " + m.Title)
		}

		// Iteramos selection hasta encontar el primer resultado de peli (su href contiene /title/)??
		var link string
		var title string

		selection.EachWithBreak(func(i int, selection *goquery.Selection) bool {
			title = selection.First().Text()
			link, _ = selection.Attr("href")
			if title == m.OriginalTitle {
				return false
			}
			return true
		})

		m.ImdbUrl = IMDBUrl + link

		m.ImdbId = link[len("/title/") : len(link)-1]

		if doc, err = goquery.NewDocument(m.ImdbUrl); err != nil {
			log.Fatal(err)
		} else {
			m.Genre = doc.Find("[itemprop=genre]").Eq(0).Text()
			m.ImdbRating = doc.Find("[itemprop=aggregateRating]").Find("[itemprop=ratingValue]").Text()
			m.Director = doc.Find("[itemprop=director]").Find("[itemprop=name]").Text()
			m.Duration = doc.Find("[itemprop=duration]").Last().Text()
		}
	}
}

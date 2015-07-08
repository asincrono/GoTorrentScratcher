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

	"github.com/PuerkitoBio/goquery"
	//"net/rpc/jsonrpc"
)

const (
	OMDBApiUrl   = "http://www.omdbapi.com"
	OMDBApiQuery = "?t=%s&type=movie"

	IMDBUrl           = "http://www.imdb.com"
	IMDBQuery         = "find?q=%s&s=all"
	IMDBAdvancedQuery = "search/title?production_status=released&sort=year,desc&title=%s&title_type=feature&view=simple"
)

//{"Title":"Love",
//"Year":"2015",
//"Rated":"N/A",
//"Released":"15 Jul 2015",
//"Runtime":"130 min",
//"Genre":"Drama",
//"Director":"Gaspar Noé",
//"Writer":"Gaspar Noé",
//"Actors":"Gaspar Noé, Karl Glusman, Aomi Muyock, Klara Kristin",
//"Plot":"A sexual melodrama about a boy and a girl and another girl. It's a love story, which celebrates sex in a joyous way.",
//"Language":"English",
//"Country":"France, Belgium",
//"Awards":"1 nomination.",
//"Poster":"http://ia.media-imdb.com/images/M/MV5BMTQzNDUwODk5NF5BMl5BanBnXkFtZTgwNzA0MDQ2NTE@._V1_SX300.jpg",
//"Metascore":"54",
//"imdbRating":"7.0",
//"imdbVotes":"122",
//"imdbID":"tt3774694",
//"Type":"movie","Response":"True"}

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
	Torrents    map[string]*torrent.Torrent

	updated bool
}

func (m *Movie) IsUpdated() bool {
	return m.updated
}

func (m *Movie) setUpdated() {
	m.updated = true
}

func (m *Movie) GetRating() (rating float32, err error) {
	val, err := strconv.ParseFloat(m.Rating, 32)
	rating = float32(val)
	return
}

func (m *Movie) GetMetascore() (rating int, err error) {
	val, err := strconv.ParseInt(m.Metascore, 10, 0)
	rating = int(val)
	return
}

func (m *Movie) GetImdbRating() (rating float32, err error) {
	val, err := strconv.ParseFloat(m.ImdbRating, 32)
	rating = float32(val)
	return
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

	url := strings.Join([]string{OMDBApiUrl, query}, "/")
	if res, err := http.Get(url); err != nil {
		log.Fatal(err)
	} else {
		defer res.Body.Close()

		rawJson, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		var jsonMovie interface{}

		if err := json.Unmarshal(rawJson, m); err != nil {
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				m.updated = true
			} else {
				m.updated = false
			}
		} else {
			m.updated = false
		}

		fmt.Printf("%+v\n", jsonMovie)
	}
}

//    public Movie enrichMovieWithImdbAPI(Movie movie) {
//        try {
//            String url = "http://www.omdbapi.com/?t={title}&type=movie";
//            String title = movie.getOriginalTitle() != null ? movie.getOriginalTitle() : movie.getTitle();
//            url = url.replace("{title}", java.net.URLEncoder.encode(title, "UTF-8"));
//            JsonNode imdb = om.readTree(new URL(url));
//            if (imdb.get("Error") != null) {
//                log.warn("IMDB API 404: " + movie.getTitle());
//            } else {
//                movie.setYear(imdb.get("Year").asText());
//                movie.setGenre(imdb.get("Genre").asText());
//                movie.setRating(imdb.get("imdbRating").asDouble());
//                movie.setImdbId(imdb.get("imdbID").asText());
//            }
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

	fmt.Println("URL: ", url)

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

			fmt.Println("Title:", title)
			fmt.Println("Link:", link)

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

//public Movie enrichMovieWithImdbSearch(Movie movie) {
//        try {
//            String url = "http://www.imdb.com/find?q={title}&s=all";
//            String title = movie.getOriginalTitle() != null ? movie.getOriginalTitle() : movie.getTitle();
//            url = url.replace("{title}", java.net.URLEncoder.encode(title, "UTF-8"));
//            Document doc = Jsoup.connect(url).get();
//            Elements results = doc.select(".result_text a");
//            if (results.size() == 0) {
//                log.warn("IMDB search 404: " + movie.getTitle());
//                return movie;
//            }
//            String link = results.first().attr("href");
//            String imdbId = link.substring("/title/".length(), link.indexOf("?") - 1);
//            movie.setImdbId(imdbId);
//            url = "http://www.imdb.com" + link;
//            doc = Jsoup.connect(url).get();
//            movie.setGenre(doc.select("[itemprop=genre]").eq(0).text());
//            String rating = doc.select("[itemprop=aggregateRating] [itemprop=ratingValue]").text();
//            if (rating.isEmpty() == false) {
//                movie.setRating(Double.valueOf(rating.replace(',', '.')));
//            }
//        } catch (IOException ex) {
//            log.warn(ex.getMessage());
//        }
//        return movie;
//    }

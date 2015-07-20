// main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"movie"
	"net/http"
	"torrent"

	"golang.org/x/net/html"
)

const (
	FilmAffinityURL = "http://www.filmaffinity.com"
	IMDBURL         = "http://www.imdb.com/"
	EliteTorrentURL = "http://www.elitetorrent.net"
	CategoriaHDRIP  = "categoria/13/peliculas-hdrip"
	ModeList        = "modo:listado"
	OrderScore      = "orden:valoracion"
	Page            = "pag:"
)

func showNodeInfo(node *html.Node) {
	fmt.Println("Node data: ", node.Data)
	if node.Attr != nil {
		for _, attr := range node.Attr {
			fmt.Println("Namespace: ", attr.Namespace)
			fmt.Println("Key: ", attr.Key)
			fmt.Println("Value: ", attr.Val)
		}
	}
}

func getRecordPaths(page uint) (paths []string) {
	url := strings.Join([]string{EliteTorrentURL, CategoriaHDRIP, ModeList, OrderScore, Page}, "/")
	url = fmt.Sprintf("%s%d", url, page)

	fmt.Println("(getRecordPaths) url = ", url)

	if res, err := http.Get(url); err != nil {
		log.Fatal(err)
	} else {
		doc, err := goquery.NewDocumentFromResponse(res)

		if err != nil {
			log.Fatal(err)
		}

		selection := doc.Find("a.nombre")

		paths = make([]string, selection.Length())

		selection.Each(func(i int, s *goquery.Selection) {
			paths[i], _ = s.Attr("href")
		})
	}

	return
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

func getMovie(path string) (m movie.Movie) {
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

		m.Title = movie.CleanTitle(title)
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

	return
}

func main() {
	var imdb bool
	var filmaffinity bool
	var omdb bool
	var overwrite bool
	var iniPage uint
	var finalPage uint

	var outFileName string

	flag.BoolVar(&imdb, "imdb", false, "Se intentará obtener la información de la película de IMDB.")
	flag.BoolVar(&filmaffinity, "filmaffinity", false, "Se intentará obtener la información de la película de Filmaffinity.")
	flag.BoolVar(&omdb, "omdb", true, "Se intentará obtener la información de la película de OMDB.")

	flag.BoolVar(&overwrite, "overwrite", false, "Se sobreescribirán los datos de la película con cada intento de obtención de información.")

	flag.UintVar(&iniPage, "ip", 1, "Página desde la que empezar a obtener películas.")
	flag.UintVar(&finalPage, "fp", 0, "Página hasta la que obtener películas.")

	flag.StringVar(&outFileName, "o", "movies", "Nombre del fichero (.json) de salida.")

	flag.Parse()

	var m movie.Movie

	var page uint
	var paths []string
	var jsonMovieRaw []byte

	//fileName := ".\\movies.json"

	fileName := "./" + outFileName + ".json"

	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		log.Fatal(err)
	}

	//	m.OriginalTitle = "Love"
	//	m.EnrichWithFilmAffinity(true)
	//	fmt.Printf("%+v\n", m)

	page = iniPage
	for {
		paths = getRecordPaths(page)
		if len(paths) == 0 {
			break
		}

		for _, path := range paths {
			m = getMovie(path)

			if imdb {
				m.EnrichWithImdbSearch(overwrite)
			}

			if filmaffinity {
				m.EnrichWithFilmAffinity(overwrite)
			}

			if omdb {
				m.EnrichWithOmdbApi()
			}

			jsonMovieRaw, _ = json.Marshal(m)
			fmt.Fprintln(file, string(jsonMovieRaw))
		}

		page += 1

		fmt.Println("Page:", page)

		if finalPage != 0 && page >= finalPage {
			break
		}
	}
}

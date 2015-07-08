// main.go
package main

import (

	//"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	//	"github.com/neocortical/gsoup"
	//	"io/ioutil"

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

func getRecordPaths(page uint16) (paths []string) {
	url := strings.Join([]string{EliteTorrentURL, CategoriaHDRIP, ModeList, OrderScore, Page}, "/")
	url = fmt.Sprintf("%s%d", url, page)

	fmt.Println("url = ", url)

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
			fmt.Printf("Path (%d): %s.\n", i, paths[i])
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

func getMovie(path string) (movie movie.Movie) {
	url := EliteTorrentURL + path
	log.Println("Retrieving", path+".")

	if res, err := http.Get(url); err != nil {
		log.Fatal(err)
	} else {

		doc, err := goquery.NewDocumentFromResponse(res)

		if err != nil {
			log.Fatal(err)
		}

		movie.Url = url

		title := doc.Find("#box-ficha > h2").Text()

		movie.Title = removeParenthesesAndBracketsContent(title)

		movie.Description = doc.Find("p.descrip").Eq(1).Text()

		movie.Rating = doc.Find("span.valoracion").Text()

		fmt.Println("Movie Rating: ", movie.Rating)

		imgName, _ := doc.Find("img.imagen_ficha").Attr("src")

		movie.Image = EliteTorrentURL + imgName

		var torrent torrent.Torrent

		torrent.Magnet, _ = doc.Find("a[href^=magnet]").Attr("href")
		torrent.Filesize = doc.Find("dl.info-tecnica dd").Eq(3).Text()

		seedsClientsText := doc.Find("div.ppal").Text()

		seedsClientsArr := strings.Fields(seedsClientsText)

		for i, str := range seedsClientsArr {
			fmt.Println(i, ":", str)
		}

		seeds, _ := strconv.Atoi(seedsClientsArr[1])
		peers, _ := strconv.Atoi(seedsClientsArr[4])

		torrent.Seeds = uint16(seeds)
		torrent.Peers = uint16(peers)

		fmt.Println("Seeds:", seeds)
		fmt.Println("Clients:", peers)

		fmt.Println(torrent)
		movie.AddTorrent("720p", &torrent)
	}

	return
}

func main() {

	str := "/title/tt3774694/?ref_=fn_al_tt_1"
	imdbId := str[len("/title/") : strings.Index(str, "?")-1]
	fmt.Println("ImdbId: ", imdbId)

	i := 0
	for {
		fmt.Println("Iter: ", i)
		if i >= 10 {
			break
		} else {
			i += 1
		}
	}

	var m movie.Movie

	m.OriginalTitle = "Love"

	m.EnrichWithImdbSearch()

	fmt.Printf("%+v\n", m)
	//url := "http://www.omdbapi.com/?t=Love&y=2015&plot=short&r=json"

	//	paths := getRecordPaths(1)

	//	movies := make([]movie.Movie, len(paths))

	//	for i, path := range paths {
	//		movies[i] = getMovie(path)
	//		movies[i].EnrichWithOmdbApi()
	//		fmt.Printf("%+v\n", movies[i])
	//	}

	//	if res, err := http.Get(url); err != nil {
	//		log.Fatal("Omdbapi didn't return any data:", err)
	//	} else {
	//		defer res.Body.Close()

	//		var m movie.Movie

	//		jsonRaw, err := ioutil.ReadAll(res.Body)

	//		if err != nil {
	//			log.Fatal(err)
	//		}

	//		if err = json.Unmarshal(jsonRaw, &m); err != nil {
	//			if _, ok := err.(*json.UnmarshalTypeError); ok {
	//				log.Print(err)
	//			} else {
	//				log.Fatal(err)
	//			}
	//		}

	//		fmt.Printf("%+v\n", m)
	//	}

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

}

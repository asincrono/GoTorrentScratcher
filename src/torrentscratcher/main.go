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

	"golang.org/x/net/html"
	"movie"
	"net/http"
	"torrent"
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
	recordPaths := getRecordPaths(1)
	//	for _, path := range recordPaths {
	//		movie := getMovie(path)
	//		fmt.Println(movie)
	//	}

	movie := getMovie(recordPaths[0])

	fmt.Println(movie)

}

/*
public Movie getMovie(String path) throws IOException {
        String url = "http://www.elitetorrent.net{path}";
        url = url.replace("{path}", path);
        log.debug("Retrieving " + path + ".");
        Document doc = Jsoup.connect(url).get();

        Movie movie = new Movie();
        String title = doc.select("#box-ficha > h2").text();
        // strip parentheses: http://stackoverflow.com/questions/1138552/replace-string-in-parentheses-using-regex
        title = title
                .replaceAll("\\([^\\(]*\\)", "")
                .replaceAll("\\[[^\\(]*\\]", "")
                .replaceAll("aka$", "")
                .trim();
        movie.setTitle(title);
        movie.setUrl(url);
        movie.setDescription(doc.select("p.descrip").eq(1).text());
        movie.setType("movie");
        movie.setImage("http://www.elitetorrent.net/" + doc.select("img.imagen_ficha").attr("src"));

        Torrent torrent = new Torrent();
        torrent.setMagnet(doc.select("a[href^=magnet]").attr("href"));
        torrent.setFilesize(doc.select("dl.info-tecnica dd").eq(3).text());
        movie.getTorrents().put("720p", torrent);

        return movie;
    }
*/

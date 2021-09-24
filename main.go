package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/goodsign/monday"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Concert struct {
	Artist   string    `bson:"artist,omitempty"`
	Location string    `bson:"location,omitempty"`
	Date     time.Time `bson:"date,omitempty"`
	Link     string    `bson:"link,omitempty"`
	Comment  string    `bson:"comment,omitempty"`
}

type concertCrawler interface {
	getConcerts() []Concert
}

type helsinkiCrawler struct {
	url string
}

func (hc helsinkiCrawler) getConcerts() []Concert {
	concerts := []Concert{}
	//url := "https://www.helsinkiklub.ch/"
	res, err := http.Get(hc.url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	//io.Copy(os.Stdout, res.Body)
	z := html.NewTokenizer(res.Body)
	var currentConcert Concert
	var previousToken, token html.Token
	token = html.Token{}
	var day, month string
	//notYetAdded := true
	parse := true
	for parse {
		tokenType := z.Next()
		previousToken = token
		token = z.Token()
		if tokenType == html.ErrorToken {
			break
		}
		if tokenType == html.StartTagToken {
			if token.DataAtom == atom.Div {
				for _, attr := range token.Attr {
					if attr.Key == "class" && attr.Val == "event" {
						if currentConcert.Artist != "" {
							concerts = append(concerts, currentConcert)
						}
						currentConcert = Concert{
							Location: "Helsinki",
							Link:     hc.url}
					}
					if attr.Key == "id" && attr.Val == "col2" {
						parse = false
						break
					}
				}
			}
		}
		if tokenType == html.TextToken {
			for _, attr := range previousToken.Attr {
				if attr.Key == "class" {
					switch attr.Val {
					case "top":
						currentConcert.Artist = html.UnescapeString(token.String())
					case "day":
						day = token.String()
					case "month":
						month = token.String()
						year := time.Now().Year() // obviously not always correct. Unfortunatly, the year is not easily extractable
						loc, _ := time.LoadLocation("Europe/Berlin")
						layout := "2 January 2006 15:00"
						d := fmt.Sprintf("%s %s %d 20:00", day, month, year)
						t, err := monday.ParseInLocation(layout, d, loc, monday.LocaleDeDE)
						if err != nil {
							log.Fatalf("Couldn't parse date %s: %v", d, err)
						}
						currentConcert.Date = t
					case "addition":
						currentConcert.Comment = html.UnescapeString(token.String())
						// sometimes the year of a concert can be found in the comment.
						re := regexp.MustCompile("20[0-9]{2}")
						match := re.FindString(currentConcert.Comment)
						if len(match) > 0 {
							d := currentConcert.Date
							year, _ := strconv.Atoi(match) // we ignore the error because the regex ensures that it's an int.
							currentConcert.Date = time.Date(int(year), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
						}
					}
				}
			}
		}
	}
	concerts = append(concerts, currentConcert)
	return concerts
}

func writeConcertsToMongoDB(c concertCrawler) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017/croncert"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	croncertDatabase := client.Database("croncert")
	concertsCollection := croncertDatabase.Collection("concerts")
	opts := options.Replace().SetUpsert(true)
	for _, concert := range c.getConcerts() {
		_, err = concertsCollection.ReplaceOne(ctx, concert, concert, opts)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	hc := helsinkiCrawler{url: "https://www.helsinkiklub.ch/"}
	// for _, concert := range c {
	// 	fmt.Printf("Artist: %v,\nLocation: %v,\nDate: %v,\nLink: %v,\nComment: %v\n\n",
	// 		concert.Artist, concert.Location, concert.Date, concert.Link, concert.Comment)
	// }
	writeConcertsToMongoDB(hc)
}

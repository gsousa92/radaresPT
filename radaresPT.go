package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly"
	_ "github.com/mattn/go-sqlite3"
)

type SpeedControl struct {
	district         string
	created_datetime string
	location         string
}

func createdDatetimeToTimestamp(createdDatetime string) int64 {
	layout := "02/01/2006 15:04:05"

	parsedTime, err := time.Parse(layout, createdDatetime)
	if err != nil {
		panic(err)
	}

	return parsedTime.Unix()
}

func sanitizeLocationString(location string) string {
	notesStartIndex := strings.Index(location, "[")
	notesEndIndex := strings.Index(location, "]")

	// no notes to remove
	if !(notesStartIndex == -1 || notesEndIndex == -1) {
		location = location[:notesStartIndex]
	}

	notesStartIndex = strings.Index(location, "LOCALIZAÇÃO APROXIMADA")
	if notesStartIndex != -1 {
		location = location[:notesStartIndex]
	}

	notesStartIndex = strings.Index(location, "Editado pela Administração por um dos motivos:")
	if notesStartIndex != -1 {
		location = location[:notesStartIndex]
	}

	return location
}

func fetch_last_speed_controls() []SpeedControl {
	var sc []SpeedControl


	// TODO swap this by chromedp (example: https://www.zenrows.com/blog/web-scraping-golang#scraping-dynamic-content)
	// to interact with the page by scrolling down and triggering lazy loading to get more results
	c := colly.NewCollector()

	c.OnHTML(".panel.panel-default ", func(e *colly.HTMLElement) {
		speed_control := SpeedControl{
			district:         e.ChildText(".panel-body > h4"),
			created_datetime: e.ChildText(".panel-heading > p"),
			location:         sanitizeLocationString(e.ChildText(".panel-body > p.lead")),
		}

		sc = append(sc, speed_control)

	})
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.Visit("https://temporeal.radaresdeportugal.pt/")

	return sc
}

func main() {
	var err error
	var mostRecentCreatedDatetime sql.NullInt64

	db, err := sql.Open("sqlite3", "file:radaresPT.db")
	if err != nil {
		panic(err)
	}

	err = db.QueryRow("SELECT MAX(created_datetime) FROM speed_controls").Scan(&mostRecentCreatedDatetime)
	if err != nil {
		panic(err)
	}

	emptyTable := true
	if mostRecentCreatedDatetime.Valid {
		// if most recent timestamp comes NULL, we consider that the table empty
		emptyTable = false
	}

	speedControls := fetch_last_speed_controls()

	valueStrings := make([]string, 0, len(speedControls))
	valueArgs := make([]interface{}, 0, len(speedControls)*3)

	for _, sc := range speedControls {
		created_datetime := createdDatetimeToTimestamp(sc.created_datetime)

		if !emptyTable && created_datetime > mostRecentCreatedDatetime.Int64 {
			valueStrings = append(valueStrings, "(?, ?, ?)")
			valueArgs = append(valueArgs, sc.district)
			valueArgs = append(valueArgs, created_datetime)
			valueArgs = append(valueArgs, sc.location)
		}
	}

	if len(valueStrings) > 0 {
		query := fmt.Sprintf("INSERT INTO speed_controls (district, created_datetime, location) VALUES %s", strings.Join(valueStrings, ","))

		_, err = db.Exec(query, valueArgs...)
		if err != nil {
			panic(err)
		}
	}

}

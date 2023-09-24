package main

import (
	"database/sql"
	"fmt"
	"github.com/gocolly/colly"
	_ "github.com/mattn/go-sqlite3"
	"strings"
)

type SpeedControl struct {
	district         string
	created_datetime string
	location         string
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

func main() {
	var err error
	var version string

	db, err := sql.Open("sqlite3", "file:radaresPT.db")

	if err != nil {
		panic(err)
	}

	fmt.Println(version)

	c := colly.NewCollector()

	var speed_controls []SpeedControl

	// Find and visit all links
	c.OnHTML(".panel.panel-default ", func(e *colly.HTMLElement) {
		speed_control := SpeedControl{
			district:         e.ChildText(".panel-body > h4"),
			created_datetime: e.ChildText(".panel-heading > p"),
			location:         sanitizeLocationString(e.ChildText(".panel-body > p.lead")),
		}

		speed_controls = append(speed_controls, speed_control)

	})
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.Visit("https://temporeal.radaresdeportugal.pt/")

	valueStrings := make([]string, 0, len(speed_controls))
	valueArgs := make([]interface{}, 0, len(speed_controls)*3)

	for _, sc := range speed_controls {
		valueStrings = append(valueStrings, "(?, ?, ?)")
		valueArgs = append(valueArgs, sc.district)
		valueArgs = append(valueArgs, sc.created_datetime)
		valueArgs = append(valueArgs, sc.location)
	}
	query := fmt.Sprintf("INSERT INTO speed_controls (district, created_datetime, location) VALUES %s", strings.Join(valueStrings, ","))

	_, err = db.Exec(query, valueArgs...)
	if err != nil {
		panic(err)
	}
}

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
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

	// initializing a chrome instance
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	var nodes []*cdp.Node
	chromedp.Run(ctx,
		chromedp.Navigate("https://temporeal.radaresdeportugal.pt/"),
		chromedp.Nodes(".panel.panel-default", &nodes, chromedp.ByQueryAll),
	)

	for _, node := range nodes {
		var district, created_datetime, location string
		chromedp.Run(ctx,
			chromedp.Text(".panel-body > h4", &district, chromedp.ByQuery, chromedp.FromNode(node), chromedp.AtLeast(0)),
			chromedp.Text(".panel-heading > p:not(.pull-right)", &created_datetime, chromedp.ByQuery, chromedp.FromNode(node), chromedp.AtLeast(0)),
			chromedp.Text(".panel-body > p.lead", &location, chromedp.ByQuery, chromedp.FromNode(node), chromedp.AtLeast(0)),
		)

		// For now ignore speed control if no location provided
		if location == "" {
			continue
		}

		speed_control := SpeedControl{
			district:         district,
			created_datetime: created_datetime,
			location:         sanitizeLocationString(location),
		}

		sc = append(sc, speed_control)

	}
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
		created_datetime := createdDatetimeToTimestamp(strings.TrimSpace(sc.created_datetime))

		// If the new entry is older than the most recent saved speed control, no need to save it,
		// because it's outdated info
		if !emptyTable && created_datetime < mostRecentCreatedDatetime.Int64 {
			break
		}

		valueStrings = append(valueStrings, "(?, ?, ?)")
		valueArgs = append(valueArgs, sc.district)
		valueArgs = append(valueArgs, created_datetime)
		valueArgs = append(valueArgs, sc.location)
	}

	if len(valueStrings) > 0 {
		query := fmt.Sprintf("INSERT INTO speed_controls (district, created_datetime, location) VALUES %s", strings.Join(valueStrings, ","))

		_, err = db.Exec(query, valueArgs...)
		if err != nil {
			panic(err)
		}
	}

}

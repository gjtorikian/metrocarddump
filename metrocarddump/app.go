package metrocarddump

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/runner"
	"github.com/gobuffalo/packr"

	"fmt"

	"github.com/codegangsta/cli"
)

const (
	easypayURL = "https://www.easypaymetrocard.com/vector/static/accounts/index.shtml"

	csvBoothCol = 1
	csvLatCol   = 5
	csvLongCol  = 6
)

type Ride struct {
	DateTime  string  `json:"dateTime"`
	Location  string  `json:"location"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
	Transport string  `json:"transport"`
}

func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = "metrocarddump"
	app.Usage = "Dump all of your EasyPay MTA rides into a JSON file."
	app.Version = "0.0.8"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "Print debug statements",
		},
		cli.BoolFlag{
			Name:  "skip, s",
			Usage: "Skips stations for which a lat/long could not be found",
		},
		cli.BoolFlag{
			Name:  "trim, t",
			Usage: "Removes all information except for lat/long (for privacy)",
		},
	}
	app.Action = func(c *cli.Context) {
		run(c)
	}

	return app
}

var run = func(cliCtxt *cli.Context) {
	debugMode := cliCtxt.Bool("debug")
	skipMissing := cliCtxt.Bool("skip")
	trimData := cliCtxt.Bool("trim")

	var err error

	// create context
	ctxt, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create chrome instance
	c, err := chromedp.New(ctxt, chromedp.WithRunnerOptions(
		runner.RemoteDebuggingPort(9222),
	))
	if err != nil {
		log.Fatal(err)
	}

	// run task list
	var rides []Ride = make([]Ride, 0)
	err = c.Run(ctxt, navigate(ctxt, c, debugMode, &rides))
	if err != nil {
		log.Fatal(err)
	}

	// shutdown chrome
	err = c.Shutdown(ctxt)
	if err != nil {
		log.Fatal(err)
	}

	// wait for chrome to finish
	err = c.Wait()
	if err != nil {
		log.Fatal(err)
	}

	writeResults(rides, skipMissing, trimData)
}

func writeResults(rides []Ride, skipMissing bool, trimData bool) {
	var missingStations []string
	var modifiedRides []Ride
	var err error

	box := packr.NewBox("./static")
	csvFile, err := box.FindString("geocoded.csv")
	if err != nil {
		log.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader(csvFile))
	lines, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range rides {
		// Buses have no lat/long
		if r.Transport == "Bus" {
			if !skipMissing && !trimData {
				modifiedRides = append(modifiedRides, r)
			}
			continue
		}

		s := strings.Split(r.Location, " ")
		booth := s[0]

		for i, line := range lines {
			if line[csvBoothCol] == booth {
				r.Latitude = toFloat(line[csvLatCol])
				r.Longitude = toFloat(line[csvLongCol])
				if trimData {
					modifiedRides = append(modifiedRides, Ride{"", "", r.Latitude, r.Longitude, ""})
				} else {
					modifiedRides = append(modifiedRides, r)
				}

				break
			} else if i == len(lines)-1 { // we are at the end and didn't find the booth
				if !skipMissing || !trimData {
					modifiedRides = append(modifiedRides, r)
				}
				missingStations = append(missingStations, r.Location)
			}
		}
	}

	ridesJson, err := json.MarshalIndent(modifiedRides, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	filename := fmt.Sprintf("%s_rides.json", time.Now().Format("20060102"))
	err = ioutil.WriteFile(filename, ridesJson, 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("wrote data to %s\n", filename)

	if len(missingStations) > 0 {
		fmt.Println("\n*** It's no one's fault (except the MTA), but I couldn't find locations for these stations:")
		for _, s := range missingStations {
			fmt.Println(s)
		}
		fmt.Print("\n(Repeated lines represent each time you rode on it.)")
	}
}

func toFloat(s string) float32 {
	value, err := strconv.ParseFloat(s, 32)
	if err != nil {
		log.Fatal(err)
	}

	return float32(value)
}

func navigate(ctxt context.Context, c *chromedp.CDP, debugMode bool, rides *[]Ride) chromedp.Tasks {
	var dropdown []*cdp.Node

	c.Run(ctxt, chromedp.Navigate(easypayURL))

	fmt.Println("starting scrape timer...")

	c.Run(ctxt, chromedp.Navigate(easypayURL))

	c.Run(ctxt, chromedp.Sleep(10*time.Second)) // give folks time to enter password, etc

	fmt.Println("scraping...")

	return chromedp.Tasks{
		chromedp.WaitVisible(`#HStatementPeriod`, chromedp.ByID),
		chromedp.Nodes(`//select[@id="HStatementPeriod"]/option`, &dropdown),
		chromedp.ActionFunc(func(context.Context, cdp.Executor) error {
			// first option is the dropdown name
			_, dropdown = dropdown[0], dropdown[1:]

			// navigate to each item in the dropdown menu
			for _, n := range dropdown {
				var url string
				url = n.AttributeValue("value")
				c.Run(ctxt, parse(ctxt, c, url, debugMode, rides))
			}

			return nil
		}),
	}
}

func parse(ctxt context.Context, c *chromedp.CDP, url string, debugMode bool, rides *[]Ride) chromedp.Tasks {
	var dateNodes []*cdp.Node
	var date string
	var locationNodes []*cdp.Node
	var location string
	var vehicleNodes []*cdp.Node
	var vehicle string

	var nextLink []byte
	var nextNode []*cdp.Node

	if debugMode {
		log.Print(fmt.Sprintf("checking %s\n", url))
	}

	return chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.WaitVisible(`#StatementTable`, chromedp.ByID),
		chromedp.Nodes(`//table[@id="StatementTable"]/tbody[1]/tr/td[2]`, &dateNodes),
		chromedp.Nodes(`//table[@id="StatementTable"]/tbody[1]/tr/td[4]`, &locationNodes),
		chromedp.Nodes(`//table[@id="StatementTable"]/tbody[1]/tr/td[5]`, &vehicleNodes),
		chromedp.ActionFunc(func(_ context.Context, e cdp.Executor) error {
			// first row is garbage header data
			_, dateNodes = dateNodes[0], dateNodes[1:]
			_, locationNodes = locationNodes[0], locationNodes[1:]
			_, vehicleNodes = vehicleNodes[0], vehicleNodes[1:]

			for i, d := range dateNodes {
				c.Run(ctxt, chromedp.Text(locationNodes[i].FullXPath(), &location))
				// if you place money on a card, its location returns blank; ignore this meta-row
				if len(strings.TrimSpace(location)) == 0 {
					continue
				}
				c.Run(ctxt, chromedp.Text(d.FullXPath(), &date))
				c.Run(ctxt, chromedp.Text(vehicleNodes[i].FullXPath(), &vehicle))

				// starts with "Ride: "
				vehicle = vehicle[6:]

				*rides = append(*rides, Ride{date, location, 0, 0, vehicle})
			}

			// detect Next link; cannot for the life of me find a simpler way in chromedp
			// to just check for a node's existence
			err := chromedp.EvaluateAsDevTools(`$x('//table[@id="StatementTable"]//a[text() = "Next"]/node()')`, &nextLink).Do(ctxt, e)
			if err != nil {
				log.Fatal(err)
			}

			// [{}] if found, but [] if not
			// like I said, I can't figure this out.
			if len(nextLink) > 2 {
				c.Run(ctxt, chromedp.Nodes(`//table[@id="StatementTable"]//a[text() = "Next"]`, &nextNode))
				c.Run(ctxt, parse(ctxt, c, nextNode[0].AttributeValue("href"), debugMode, rides))
			}

			return nil
		}),
	}
}

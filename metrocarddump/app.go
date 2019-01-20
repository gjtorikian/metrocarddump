package metrocarddump

import (
	"context"
	"log"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/runner"

	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/codegangsta/cli"
)

const (
	easypayURL = "https://www.easypaymetrocard.com/vector/static/accounts/index.shtml"
)

type Ride struct {
	DateTime  string `json:"date_time"`
	Location  string `json:"location"`
	Transport string `json:"transport"`
}

func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = "metrocarddump"
	app.Usage = "Dump all of your EasyPay MTA rides into a JSON file."
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "print debug statements",
		},
	}
	app.Action = func(c *cli.Context) {
		run(c)
	}

	return app
}

var run = func(cliCtxt *cli.Context) {
	debugOn := !cliCtxt.Bool("debug")

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
	err = c.Run(ctxt, navigate(ctxt, c, debugOn, &rides))
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

	ridesJson, err := json.MarshalIndent(rides, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	filename := "rides.json"
	err = ioutil.WriteFile(filename, ridesJson, 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("wrote data to %s\n", filename)
}

func navigate(ctxt context.Context, c *chromedp.CDP, debugOn bool, rides *[]Ride) chromedp.Tasks {
	var dropdown []*cdp.Node
	return chromedp.Tasks{
		chromedp.Navigate(easypayURL),
		chromedp.WaitVisible(`#HStatementPeriod`, chromedp.ByID),
		chromedp.Nodes(`//select[@id="HStatementPeriod"]/option`, &dropdown),
		chromedp.ActionFunc(func(context.Context, cdp.Executor) error {
			for _, n := range dropdown {
				var url string
				url = n.AttributeValue("value")
				// navigate to each item in the dropdown menu
				if debugOn {
					fmt.Printf("checking %s\n", url)
				}
				c.Run(ctxt, parse(ctxt, c, url, debugOn, rides))
			}

			return nil
		}),
	}
}

func parse(ctxt context.Context, c *chromedp.CDP, url string, debugOn bool, rides *[]Ride) chromedp.Tasks {
	var dateNodes []*cdp.Node
	var date string
	var locationNodes []*cdp.Node
	var location string
	var vehicleNodes []*cdp.Node
	var vehicle string

	var nextLink []*cdp.Node

	return chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.WaitVisible(`#StatementTable`, chromedp.ByID),
		chromedp.Nodes(`//table[@id="StatementTable"]/tbody[1]/tr/td[2]`, &dateNodes),
		chromedp.Nodes(`//table[@id="StatementTable"]/tbody[1]/tr/td[4]`, &locationNodes),
		chromedp.Nodes(`//table[@id="StatementTable"]/tbody[1]/tr/td[5]`, &vehicleNodes),
		chromedp.ActionFunc(func(context.Context, cdp.Executor) error {
			// first row is garbage header data
			_, dateNodes = dateNodes[0], dateNodes[1:]
			_, locationNodes = locationNodes[0], locationNodes[1:]

			for i, d := range dateNodes {
				c.Run(ctxt, chromedp.Text(d.FullXPath(), &date))
				c.Run(ctxt, chromedp.Text(locationNodes[i].FullXPath(), &location))
				c.Run(ctxt, chromedp.Text(vehicleNodes[i].FullXPath(), &vehicle))

				*rides = append(*rides, Ride{date, location, vehicle})
			}

			log.Print("I shall check for next")

			// detect Next link
			c.Run(ctxt, chromedp.Nodes(`//table[@id="StatementTable"]//a[text() = "Next"]`, &nextLink))

			log.Print("Validating...")
			if &nextLink[0] != nil {
				if debugOn {
					log.Print("Found another page!\n")
				}
				c.Run(ctxt, parse(ctxt, c, nextLink[0].AttributeValue("href"), debugOn, rides))
			}
			log.Print("No next link...")

			return nil
		}),
	}
}

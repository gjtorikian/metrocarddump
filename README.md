# metrocarddump

This program will dump all of your EasyPay MTA rides from <https://www.easypaymetrocard.com> into a JSON file.

## Installation

You will need [Google Chrome](https://www.google.com/chrome/) installed. (Sorry.)

* If you have [the Go language](https://golang.org/dl/) installed, you can install this via the CLI:

      go get github.com/gjtorikian/metrocarddump

* If you don't have Go installed, you can download a [prebuilt binary for your
platform](https://github.com/gjtorikian/metrocarddump/releases), optionally renaming it to "metrocarddump" for convenience.

## Usage

1. Run `metrocarddump`.
2. Google Chrome will pop open. Enter your account credenitals into the browser:

    ![Entering credentials](https://user-images.githubusercontent.com/64050/51440923-08d18180-1c9a-11e9-9f25-f6a534786d26.gif)

3. Click on **Statement History**. Do this within ten seconds, or the script will timeout!
4. Don't do anything.
5. Let the program do its thing.
6. When everything is done, Chrome will close, and you'll have a file called _YYYYMMDD_rides.json_.

**Note:** Unfortunately, I could not find data for some newer MTA stations, such as those off of [the Second Avenue Subway](https://en.wikipedia.org/wiki/86th_Street_(Second_Avenue_Subway)). Any missing geo-coordinates will be reported at the end of the run.

## Configuration

The `metrocarddump` bin takes arguments!

| Option | Description | Default |
| :----- | :---------- | :------ |
| `--debug`, `-d` | If `true`, prints debug statements along the way. | `false` |
| `skip`, `-s` | If `true`, skips stations for which a lat/long could not be found. | `false` |
| `trim`, `-t` | If `true`, removes all information except for lat/long (for privacy). | `false` |

## Format

The JSON returns with the following keys:

* `dateTime`: The date and time of the ride.
* `location`: The unintuitive format the MTA uses to define a station.
* `latitude`: The latitude of the station. This can be 0 if the MTA has not documented this.
* `longitude`: The longitude of the station. This can be 0 if the MTA has not documented this.
* `transport`: Whether this was a `Subway` or a `Bus` swipe.

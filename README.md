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
2. Google Chrome will pop open. Enter your account credenitals into the browser.
3. Click on ** **.
4. Don't do anything.
5. Let the program do its thing.
6. When everything is done, Chrome will close, and you'll have a file called _rides.json_.

## Configuration

The `metrocarddump` bin takes arguments!

| Option | Description | Default |
| :----- | :---------- | :------ |
| `debug` | If `true`, prints debug statements along the way. | `false` |

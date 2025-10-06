# goskyr

<div style="text-align:center"><img src="goskyr-logo.png" alt="goskyr logo" style="height: 300px; width:300px;"/></div>

[![Release](https://img.shields.io/github/release/jakopako/goskyr.svg)](https://github.com/jakopako/goskyr/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/jakopako/goskyr)](https://goreportcard.com/report/github.com/jakopako/goskyr)
![tests](https://github.com/jakopako/goskyr/actions/workflows/go-tests.yml/badge.svg?event=push)

1. [Quick Start](#quick-start)
1. [Installation](#installation)
1. [Semi-Automatic Configuration](#semi-automatic-configuration)
1. [Manual Configuration & Usage](#manual-configuration--usage)
   1. [Static fields](#static-fields)
   1. [Dynamic fields](#dynamic-fields)
   1. [Fetcher](#fetcher)
   1. [Filters](#filters)
   1. [Interaction](#interaction)
   1. [Pagination](#pagination)
   1. [Output](#output)
1. [Build ML Model for Improved Auto-Config](#build-ml-model-for-improved-auto-config)
1. [Related Projects](#related-projects)
1. [Build & Release](#build--release)
1. [Contributing](#contributing)
1. [Naming](#naming)
1. [Similar Projects](#similar-projects)

This project's goal is to make it easier to **scrape list-like structured data** from web pages. This could be a list of books from an online book store, a list of plays in a public theater, a list of newspaper articles, etc. Currently, the biggest use-case that I know of is [croncert](https://github.com/jakopako/croncert-config) which is also the main motivation behind this project.

Next to [manually configuring](#manual-configuration--usage) the scraper there is an option of (semi-)automatically generating a configuration file, see [quick start](#quick-start) and [Semi-Automatic Configuration](#semi-automatic-configuration). **Machine learning** can be leveraged to predict field names more or less accurately, see section [Build ML Model for Improved Auto-Config](#build-ml-model-for-improved-auto-config).

## Quick Start

First, [install goskyr](#installation) and then run the following steps to generate a configuration file for the scraper. The configuration file is written to the default location `config.yml`. Navigation in the interactive terminal window is done with the arrow keys, the return key and the tab key.

```bash
goskyr generate -u https://www.imdb.com/chart/top/ -D
```

Note, that different colors are used to show how 'close' certain fields are to each other in the html tree. This should help when there are multiple list-like structures on a web page and you need to figure out which fields belong together.

Next, start the scraping process. The configuration file is read from the default location `config.yml`.

```bash
goskyr scrape
```

Optionally, modify the configuration file according to your needs. For more information check out the section on [manually configuring](#manual-configuration--usage) the scraper. For a better understanding of the command line flags run

```bash
goskyr -h
```

Note that the feature to (semi-)automatically generate a configuration file is currently in an experimental stage and might not properly work in a lot of cases.

## Installation

### Arch Linux

[goskyr](https://aur.archlinux.org/packages/goskyr) is available as a package in the AUR. Install it with your favorite AUR helper (eg. `yay`):

```bash
yay -S goskyr
```

### Other

[Download](https://github.com/jakopako/goskyr/releases/latest) a prebuilt binary from the [releases page](https://github.com/jakopako/goskyr/releases), unpack and run!

Or if you have recent go compiler installed download goskyr by running

```bash
go install github.com/jakopako/goskyr@latest
```

Or clone the repository and then run with `go run main.go ...` or build it yourself.

## Semi-Automatic Configuration

As shown under [Quick Start](#quick-start) goskyr can be used to automatically extract a configuration for a given url. A number of different options are available.

```
$ goskyr generate -h
Usage: goskyr generate --url=STRING [flags]

Generate a scraper configuration file for the given URL

Flags:
  -h, --help                       Show context-sensitive help.
  -v, --version                    Print the version and exit.
  -d, --debug                      Set log level to 'debug' and store additional helpful debugging data.

  -u, --url=STRING                 The URL for which to generate the scraper configuration file.
  -m, --min-occurrence=20          The minimum number of occurrences of a certain field on an html page to be included in the suggested fields. This is needed to filter out noise.
  -D, --distinct                   If set to true only fields with distinct values will be included in the suggested fields.
  -r, --render-js                  Render javascript before analyzing the html page.
  -w, --word-lists="word-lists"    The directory that contains a number of files containing words of different languages, needed for extracting ML features.
  -M, --model-name=STRING          The name to a pre-trained ML model to infer names of extracted fields.
  -o, --stdout                     If set to true the the generated configuration will be written to stdout.
  -c, --config="./config.yml"      The file that the generated configuration will be written to.
```

A few more details on the ML part.

With the `-M` / `--model-name` flag, you can pass a reference to a ML model that suggests names for the extracted fields. Note that the model currently consists of two files that have to be named exactly the same except for the ending. The string that you have to pass to the `--model` flag has to be the filename without the ending. Check out the section on [building a ML model](#build-ml-model-for-improved-auto-config).

The flag `-w` / `--word-lists` is used to pass a the name of a directory that contains a bunch of text files with dictionary words. This is needed for feature extraction for the ML stuff. This repository contains an example of such a directory, `word-lists`, although the lists are pretty limited. Default is `word-lists`.

Note that when using machine learning & a properly trained model, the auto configuration is capable of determining what fields could be a date and what date components they contain. With that information another algorithm then tries to derive the format of the date that is needed for proper parsing. So in the best case you have to do nothing more than rename some of the fields to get the desired configuration.

Note that the machine learning feature is rather limited and might not always work well, especially since it only takes into account a fields value and not its position in the DOM. A basic model is contained in the `ml-models` directory. It uses the labels `text`, `url` and `date-component-*`. You could for instance run `goskyr generate -u https://www.schuur.ch/programm/ --model-name ml-models/knn-types-v0.4.4` which would suggest the following fields to you.

![screenshot field extraction](schuur-extract.png)

## Manual Configuration & Usage

Despite the option to automatically generate a configuration file for goskyr there are a lot more options that can be configured manually. Note that while writing and testing a new configuration it might make sense to use the `--debug` flag when running goskyr, to enable more detailed logging and have the scraped html's written to files.

A very simple configuration would look something like this:

```yml
scrapers:
  - name: LifeQuotes # The name is only for logging and scraper selection (with -s) and does not appear in the json output.
    url: "https://www.goodreads.com/quotes/tag/life"
    item: ".quote"
    fields:
      - name: "quote"
        location:
          selector: ".quoteText"
      - name: "author"
        location:
          selector: ".authorOrTitle"
```

Save this to a file, e.g. `quotes-config.yml` and run `goskyr -c quotes-config.yml` (or `go run main.go -c quotes-config.yml`) to retreive the scraped quotes as json string. The result should look something like this:

```json
[
  {
    "author": "Marilyn Monroe",
    "quote": "“I'm selfish, impatient and a little insecure. I make mistakes, I am out of control and at times hard to handle. But if you can't handle me at my worst, then you sure as hell don't deserve me at my best.”"
  },
  {
    "author": "William W. Purkey",
    "quote": "“You've gotta dance like there's nobody watching,"
  },
  ...
]
```

A more complex configuration might look like this:

```yml
scrapers:
  - name: Kaufleuten
    url: "https://kaufleuten.ch/events/kultur/konzerte/"
    item: ".event"
    fields:
      - name: "location"
        value: "Kaufleuten"
      - name: "city"
        value: "Zurich"
      - name: "type"
        value: "concert"
      - name: "title"
        location:
          selector: "h3"
          regex_extract:
            exp: "[^•]*"
            index: 0
      - name: "comment"
        can_be_empty: true
        location:
          selector: ".subtitle strong"
      - name: "url"
        type: "url"
        location:
          selector: ".event-link"
      - name: "date"
        type: "date"
        on_subpage: "url"
        components:
          - covers:
              day: true
              month: true
              year: true
              time: true
            location:
              selector: ".event-meta time"
              attr: "datetime"
            layout: ["2006-01-02T15:04:05-07:00"]
        date_location: "Europe/Berlin"
    filters:
      - field: "title"
        exp: "Verschoben.*"
        match: false
      - field: "title"
        exp: "Abgesagt.*"
        match: false
```

The result should look something like this:

```json
[
  {
    "city": "Zurich",
    "comment": "Der Schweizer Singer-Songwriter, mit Gitarre und bekannten sowie neuen Songs",
    "date": "2022-03-09T19:00:00+01:00",
    "location": "Kaufleuten",
    "title": "Bastian Baker",
    "type": "concert",
    "url": "https://kaufleuten.ch/event/bastian-baker/"
  },
  {
    "city": "Zurich",
    "comment": "Der kanadische Elektro-Star meldet sich mit neuem Album zurück",
    "date": "2022-03-13T19:00:00+01:00",
    "location": "Kaufleuten",
    "title": "Caribou",
    "type": "concert",
    "url": "https://kaufleuten.ch/event/caribou/"
  },
  ...
]
```

Basically, a config file contains a list of scrapers that each may have static and / or dynamic fields. Additionally, items can be filtered based on regular expressions and pagination is also supported. The resulting array of items is written to stdout or a file, as json string.

### Static fields

Each scraper can define a number of static fields. Those fields are the same over all returned items. For the event scraping use case this might be the location name as shown in the example above. For a static field only a name and a value need to be defined:

```yml
fields:
  - name: "location"
    value: "Kaufleuten"
```

### Dynamic fields

Dynamic fields are a little more complex as their values are extracted from the web page and can have different types. In the most trivial case it suffices to define a field name and a selector so the scraper knows where to look for the corresponding value. The quotes scraper is a good example for that:

```yml
fields:
  - name: "quote"
    type: "text" # defaults to 'text' if ommited
    location:
      selector: ".quoteText"
```

A dynamic field can have one of the following three types: `text`, `url` or `date`. The following table shows which options are available for which type.

| Option        | Type `text` | Type `url` |     Type `date`     | Default value |
| ------------- | :---------: | :--------: | :-----------------: | ------------- |
| can_be_empty  |      X      |     X      |                     | `false`       |
| components    |             |            |          X          | `[]`          |
| date_language |             |            |          X          | `"de_DE"`     |
| date_location |             |            |          X          | `"UTC"`       |
| guess_year    |             |            |          X          | `false`       |
| hide          |      X      |     X      |          X          | `false`       |
| location      |      X      |     X      | X (date components) | `[]`          |
| name          |      X      |     X      |          X          | `""`          |
| on_subpage    |      X      |     X      |          X          | `""`          |
| separator     |      X      |            |                     | `""`          |
| transform     |      X      |     X      | X (date components) | `[]`          |
| type          |      X      |     X      |          X          | `"text"`      |

#### Options explained

**`can_be_empty`**

If set to `false`, an error message will be printed for each item where this field is missing (i.e. the html node does not exist or the corresponding string is empty) and the correspondig item will not be included in the resulting list of items. If set to `true` there won't be an error message and the corresponding value will be an empty string.

**`components`**

This key contains the configuration for the different date components that are needed to extract a valid date. A list of the following form needs to be defined.

```yml
components:
  - covers: # what part of the date is covered by the element located at 'location'?
      day: bool # optional
      month: bool # optional
      year: bool # optional
      time: bool # optional
    location: # the location has the same configuration as explained under option 'location' with the exception that it is not a list but just a single location configuration.
      selector: "<selector>"
      ...
    layout: ["<layout>"] # a list of layouts that apply to this date component. Needs to be configured the "golang-way" and always in English.
  - covers:
      ...
```

The following example should give you a better idea how such the definition of `components` might actually look like.

```yml
components:
  - covers:
      day: true
    location:
      selector: ".commingupEventsList_block2"
    layout: ["02. "]
  - covers:
      month: true
    location:
      selector: ".commingupEventsList_block3"
    layout: ["January"]
  - covers:
      time: true
    location:
      selector: ".commingupEventsList_block4"
    layout: ["15Uhr04"]
```

For more details about the layout check out [this link](https://yourbasic.org/golang/format-parse-string-time-date-example/) or have a look at the numerous examples in the `concerts-config.yml`. Also note that mostly the layout list only contains one element. Only in rare cases where different events on the same site have different layouts it is necessary to define more than one layout.

**`date_language`**

The `date_language` needs to correspond to the language on the website. Note, that this doesn't matter for dates that only contain numbers. The values that are supported are the ones supported by the underlying library, [goodsign/monday](https://github.com/goodsign/monday).

**`date_location`**

`date_location` sets the time zone of the respective date.

**`guess_year`**

If set to `false` and no date component is defined that covers the year, the year of the resulting date defaults to the current year. If set to `true` and no date component is defined that covers the year, goskyr will try to be 'smart' in guessing the year. This helps if a scraped list of dates covers more than one year and/or scraped dates are not within the current year but the next. Note that there are definitely some cases where this year guessing does not yet work.

**`hide`**

This option determines whether a field should be exlcuded from the resulting items. This can be handy when you want to filter based on a field that you don't want to include in the actual items. For more information on filters checkout the [Filters](#filters) section below.

**`location`**

There are two options on how to use the `location` key. Either you define a bunch of subkeys directly under `location` or you define a list of items each containing those subkeys. The latter is useful if you want the value of a field to be juxtaposition of multiple nodes in the html tree. The `separator` option will be used to join the strings. A very simple (imaginary) example could look something like this.

```yaml
fields:
  - name: artist
    location:
      - selector: div.artist
      - selector: div.country
    separator: ", "
```

The result may look like this.

```json
[...
{"artist": "Jacob Collier, UK"},
...]
```

_Subkey: `regex_extract`_

In some cases, it might be a bit more complex to extract the desired information. Take for instance the concert scraper configuration for "Kaufleuten", shown above, more specifically the config snippet for the `title` field.

```yml
fields:
  - name: "title"
    location:
      selector: "h3"
      regex_extract:
        exp: "[^•]*"
        index: 0
        ignore_errors: false # default is false
```

This field is implicitly of type `text`. The `location` tells the scraper where to look for the field value and how to extract it. In this case the selector on its own would not be enough to extract the desired value as we would get something like this: `Bastian Baker • Konzert`. That's why there is an extra option to define a regular expression to extract a substring. Note that in this example our extracted string would still contain a trailing space which is automatically removed by the scraper. Moreover, if `ignore_errors` is set to true, the scraper would not skip the given field throwing an error but would return an empty string instead. Let's have a look at a few more examples to have a better understanding of the location configuration.

_Subkey: `child_index`_

Next, let's say we want to extract the time "20h00" from the following html snippet.

```html
<div class="col-sm-8 col-xs-12">
  <h3>Freitag, 25. Feb 2022</h3>

  <h2>
    <a href="/events/924"
      ><strong>Jacob Lee (AUS) - Verschoben</strong>
      <!--(USA)-->
    </a>
  </h2>
  <q>Singer & Songwriter</q>

  <p><strong>+ Support</strong></p>
  <i
    ><strong>Doors</strong> : 19h00 /
    <strong>Show</strong>
    : 20h00
  </i>
</div>
```

This can be achieved with the following configuration:

```yml
location:
  selector: ".col-sm-8 i"
  child_index: 3
  regex_extract:
    exp: "[0-9]{2}h[0-9]{2}"
```

Here, the selector is not enough to extract the desired string and we can't go further down the tree by using different selectors. With the `child_index` we can point to the exact string we want. A `child_index` of 0 would point to the first `<strong>` node, a `child_index` of 1 would point to the string containing "19h00", a `child_index` of 2 would point to the second `<strong>` node and finally a `child_index` of 3 points to the correct string. If `child_index` is set to -1 the first child that results in a regex match will be used. This can be useful if the `child_index` varies across different items. In the current example however, the `child_index` is always the same but the string still contains more stuff than we need which is why we use a regular expression to extract the desired substring.

_Subkey: `entire_subtree`_

This subkey, if set to `true` causes goskyr to grab all text elements under the element defined in the location's selector. It is useful when the target location contains inline tags, eg. `This is some text with a <strong>strong</strong> part.`

_Subkey: `all_nodes`_

This subkey, if set to `true` joins together all strings having the given selector. The subkey `separator` will be used as separator string. If not defined the separator is an empty string. Example:

```html
<div class="header">
  <h3 class="artist"><span class="name">Anja Schneider</span><span class="artist-info"></h3>
  <h3 class="artist"><span class="name">Steve Bug</span><span class="artist-info"></h3>
  <h3 class="artist"><span class="name">Dirty Flav</span><span class="artist-info">&nbsp;(WAD, D! Club - CH)</h3>
</div>
```

Config:

```yaml
fields:
  - name: title
    location:
      selector: .artist .name
      all_nodes: true
      separator: ", "
```

Resulting json:

```json
[...
{"title": "Anja Schneider, Steve Bug, Dirty Flav"},
...]
```

To get an even better feeling for the location configuration check out the numerous examples in the `concerts-config.yml` file.

_Subkey: `json_selector`_

If the string extracted from the web page is a json string, then you can extract data from that json based on the give `json_selector`.

_Subkey: `default`_

If no value is found with the given configuration of this `location` the value defaults to `default`.

**`name`**

The name of the respective field.

**`on_subpage`**

If set to the name of another scraped field of type `url`, goskyr will fetch the corresponding page and extract the desired data from that page.

**`separator`**

This option is only relevant if the `location` option contains a list of locations of length > 1. If it does, the extracted strings (1 per location) will be joined using the defined separator.

**`transform`**

This option allows you to transform extracted text, urls and date components. Currently, the only transform type is `regex-replace`. As the name suggests, this type allows you to replace a substring that matches the given regular expression with a user defined string. An example usage of this option would be as follows.

```yaml
- name: title
  type: text
  location:
    - selector: div.event-info.single-day:nth-child(2) > div.event-title > h3 > a
  transform:
    - type: regex-replace
      regex: regex.*
      replace: New value
```

Note, that the `transform` can also be used for date components, eg.

```yaml
- name: date
  type: date
  components:
    - covers:
        day: true
        month: true
      location:
        selector: div.col-12.col-md-3 > div.g-0.row > div.col-sm-12.p-0 > div.rhp-event-thumb > a.url > div.eventDateListTop > div.eventMonth.mb-0.singleEventDate.text-uppercase
      layout:
        - Mon, January 2
        - Mon, Jan 2
      transform:
        - type: regex-replace
          regex: Sept
          replace: Sep
```

**`type`**

This is the type of the field. As mentioned above its value can be `text`, `url` or `date`.

For a field of type `text` the value that is being extracted from the web page based on the defined location will simply be assigned to the value of the corresponding field in the output.

If a field has type `url`, the resulting value in the output will allways be a full, valid url, meaning that it will contain protocol, hostname, path and query parameters. If the web page does not provide this, goskyr will 'autocomplete' the url like a browser would. E.g. if a web page, `https://event-venue.com`, contains `<a href="/events/10-03-2023-krachstock-final-story" >` and we would have a field of type `url` that extracts this url from the href attribute the resulting value would be `https://event-venue.com/events/10-03-2023-krachstock-final-story`. Also, the `location.attr` field is implicetly set to `"href"` if not defined by the user.

A `date` field is different from a text field in that the result is a complete, valid date. Internally, this is a `time.Time` object but in the json output it is represented by a string in RFC3339 format. In order to be able to handle a lot of different cases where date information might be spread across different locations, might be formatted in different ways using different languages a date field has a list of components and some other optional settings, see table above.

As can be seen, a component has to define which part of the date it covers (at least one part has to be covered). Next, the location of this component has to be defined. This is done the same way as we defined the location for a text field string. Finally, we need to define a list of possible layouts where each layout is defined the 'go-way' as this scraper is written in go. For more details check out [this](https://yourbasic.org/golang/format-parse-string-time-date-example/) link or have a look at the numerous examples in the `concerts-config.yml` file. Note that a layout string is always in English although the date string on the scraped website might be in a different language. Also note that mostly the layout list only contains one element. Only in rare cases where different events on the same site have different layouts it is necessary to define more than one layout.

The `date_language` key needs to correspond to the language on the website. Currently, the default is `de_DE`. Note, that this doesn't matter for dates that only contain numbers. `date_location` sets the time zone of the respective date.

### Fetcher

Different ways of fetching a web page are supported. The two supported types are `static` and `dynamic`. By default a scraper uses a static fetcher, i.e. does not render any javascript. You can configure a static fetcher explicitly if you like.

```yml
fetcher:
  type: static
```

To render javascript before extracting any data from a web page, you need to use the dynamic fetcher. For this to work the `google-chrome` binary needs to be installed.

```yml
fetcher:
  type: dynamic
  page_load_wait_ms: 1000 # optional. Defaults to 2000 ms
```

If using the dynamic fetcher there are ways of interacting with the page, see below section [Interaction](#interaction).

For both types of fetcher, there is an option to customize the user agent.

```yml
fetcher:
  user_agent: "Mozilla"
```

### Interaction

If a dynamic web page does initially not load all the items it might be necessary to click some kind of 'load more' button or scroll down the page. Multiple, consecutive interactions can be configured for one page.

#### Interaction types

**`click`**

```yml
interaction:
  - type: click
    selector: .some > div.selector
    count: 1 # number of clicks. Default is 1
    delay: 2000 # milliseconds that the scraper waits after the click. Default is 500
```

**`scroll`**

```yml
interaction:
  - type: scroll # scroll to the bottom of a page
    delay: 2000 # milliseconds that the scraper waits after triggering the scroll. Default is 500
```

Note that interactions are executed before the data is scraped. Also the interaction configuration will only be respected if the dynamic fetcher is used because only in that case is the website actually run within a headless browser.

### Filters

Filters can be used to define what items should make it into the resulting list of items. A filter configuration can look as follows:

```yml
filters:
  - field: "status"
    exp: "cancelled"
    match: false
  - field: "status"
    exp: ".*(?i)(delayed).*"
    match: false
  - field: "date"
    exp: "> now" # format: <|> now|YYYY-MM-ddTHH:mm
    match: true
```

The `field` key determines to which field the expression will be applied. `exp` defines the expression and `match` determines whether the item should be included or excluded on match. Note, that as soon as there is one match for an expression that has `match` set to **false** the respective item will be excluded from the results without looking at the other filters.

The expression `exp` can be either a regular expression or a date comparison. Depending on the type of the respective `field` in the `fields` section of the configuration it has to be either one or the other. If the corresponding field is of type `date` the expression has to be a date comparison. For every other field type it has to be a regular expression.

### Pagination

If the list of items on a web page spans multiple pages pagination can be configured as follows:

```yml
paginator:
  location:
    selector: ".pagination .selector"
```

If the static fetcher is used by default the value of the `href` key is taken as url for the next page. However, you can change this and other parameters in the paginator configuration.

```yml
paginator:
  location:
    selector: ".pagination .selector"
    attr: <string>
  max_pages: <number>
```

If the dynamic fetcher is used the scraper will simulate a mouse click on the given selector to loop over the pages.

### Output

Currently, the scraped data can either be written to stdout or to a file. If you don't explicitely configure the output in the configuration file the data is written to stdout. Otherwise you have to add the following snippet to your configuration file.

```yaml
writer:
  type: file
  filepath: test-file.json
```

## Build ML Model for Improved Auto-Config

In order for the auto configuration feature to find suitable names for the extracted fields, since `v0.4.0` machine learning can be used. Goskyr allows you to extract a fixed set of features based on an existing goskyr configuration. Basically, goskyr scrapes all the websites you configured, extracts the raw text values based on the configured fields per site and then calculates the features for each extracted value, labeling the resulting vector with the field name you defined in the configuration. Currently, all features are based on the extracted text only, i.e. not on the location within the website. Checkout the `Features` struct in the `ml/ml.go` file if you want to know what exactly those features are. Extraction command:

```bash
goskyr extract -o features.csv -w word-lists -c some-goskyr-config.yml
```

Note that `-w` and `-c` are optional. The respective defaults are `word-lists` and `config.yml`. The resulting csv file can optionally be edited (eg if you want to remove or replace some labels) and consequently be used to build a ML model, like so:

```bash
goskyr train -f features.csv
```

Currently, a KNN classifier is used. The output of the above command shows the result of the training. Additionally, two files are generated, `goskyr.model` and `goskyr.class`. Both together define the model that can be used for labeling fields during auto configuration, see [Semi-Automatic Configuration](#semi-automatic-configuration).

Note that the classification will probably get better the more data you have to extract your features from. Also there might very well be cases where even a huge number of training data doesn't improve the classification results. This entire ML feature is rather experimental for now and time will tell how well it works and what needs to be improved or changed.

A real life example can be found in the [jakopako/croncert-config](https://github.com/jakopako/croncert-config) repository.

## Related Projects

The main motivation to start this project was a website idea that I wanted to implement. Currently, there are four
repositories involved in this idea. The first one is of course this one, goskyr. The other three are:

- [croncert-web](https://github.com/jakopako/croncert-web): a website that shows concerts in your area, deployed to [concertcloud.live](https://concertcloud.live).
- [croncert-config](https://github.com/jakopako/croncert-config): a repository that contains a big configuration file for
  goskyr, where all the concert venue websites that are part of [concertcloud.live](https://concertcloud.live) are configured. If you're interested, check out this repository to find out how to add new concert locations and to make yourself more familiar with how to use goskyr.
- [event-api](https://github.com/jakopako/event-api): an API to store and fetch concert info, that serves as backend for
  [concertcloud.live](https://concertcloud.live).

## Build & Release

To build and release a new version of goskyr [Goreleaser](https://goreleaser.com/) is used, also see [Quick Start](https://goreleaser.com/quick-start/).

1. Make a "dry-run" release to see if it works using the release command:

```bash
make release-dry-run
```

1. Make sure you have a file called `.release-env` containing the github token.

```bash
GITHUB_TOKEN=YOUR_GH_TOKEN
```

1. Create a tag and push it to GitHub

```bash
git tag -a v0.1.5 -m "new features"
git push origin v0.1.5
```

1. Run GoReleaser at the root of this repository:

```bash
make release
```

## Contributing

Feel free to contribute in any way you want! Help is always welcome.

## Naming

Go Scraper > Go Scr > Go Skyr > goskyr

## Similar Projects

There are similar projects that might do a better job in certain cases or are more generic tools. However, on the one hand this is a personal project to make myself familiar with webscraping and Go and on the other hand goskyr supports certain features that I haven't found in any other projects. For instance, the way dates can be extracted from websites, the notion of scraping information from subpages defined by previously at runtime extracted urls and how a website's structure can be automatically detected to decrease manual configuration effort.

Similar projects:

- [MontFerret/ferret](https://github.com/MontFerret/ferret)
- [slotix/dataflowkit](https://github.com/slotix/dataflowkit)
- [andrewstuart/goq](https://github.com/andrewstuart/goq)

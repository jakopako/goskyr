# goskyr

[![Release](https://img.shields.io/github/release/jakopako/goskyr.svg)](https://github.com/jakopako/goskyr/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/jakopako/goskyr)](https://goreportcard.com/report/github.com/jakopako/goskyr)

This project's goal is to make it easier to scrape structured data from web pages. Initially, the main use case was to extract event data from
different venue websites. However, the code has been rewritten to handle a more general use case of extracting a list of items from any website.
This could be a list of books from an online book store, a list of plays in a public theater, a list of newspaper articles, etc. Currently, information can only be extracted from static websites.

Note that there are already similar projects that probably do a much better job. However, this is more of an excercise to improve my go skills and get a
better feeling for crawling and parsing websites.

Similar projects:

- [MontFerret/ferret](https://github.com/MontFerret/ferret)
- [slotix/dataflowkit](https://github.com/slotix/dataflowkit)
- [andrewstuart/goq](https://github.com/andrewstuart/goq)

## Installation

[Download](https://github.com/jakopako/goskyr/releases/latest) a prebuilt binary from [releases page](https://github.com/jakopako/goskyr/releases), unpack and run!

Or if you have recent go compiler installed download goskyr by running

```bash
go install github.com/jakopako/goskyr@latest
```

Or clone the repository and then run with `go run main.go ...` or build it yourself.

## Configuration & Usage

A very simple configuration would look something like this:

```yml
scrapers:
  - name: LifeQuotes # The name is only for logging and scraper selection (with -single) and does not appear in the json output.
    url: "https://www.goodreads.com/quotes/tag/life"
    item: ".quote"
    fields:
      dynamic:
        - name: "quote"
          location:
            selector: ".quoteText"
        - name: "author"
          location:
            selector: ".authorOrTitle"
```

Save this to a file, e.g. `quotes-config.yml` and run `goskyr -config quotes-config.yml` (or `go run main.go -config quotes-config.yml`) to retreive the scraped quotes as json string. The result should look something like this:

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
      static:
        - name: "location"
          value: "Kaufleuten"
        - name: "city"
          value: "Zurich"
        - name: "type"
          value: "concert"
      dynamic:
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
              layout: "2006-01-02T15:04:05-07:00"
          date_location: "Europe/Berlin"
    filters:
      - field: "title"
        regex: "Verschoben.*"
        match: false
      - field: "title"
        regex: "Abgesagt.*"
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

Basically, a config file contains a list of scrapers that each may have static and / or dynamic fields. Additionally, items can be filtered based on regular expressions and pagination is also supported. The resulting array of items is return to stdout as json string. TODO: support writing other outputs, e.g. mongodb.

### Static fields

Each scraper can define a number of static fields. Those fields are the same over all returned items. For the event crawling use case this might be the location name as shown in the example above. For a static field only a name and a value need to be defined:

```yml
fields:
  static:
    - name: "location"
      value: "Kaufleuten"
```

### Dynamic fields

Dynamic fields are a little more complex as their values are extracted from the webpage and can have different types. In the most trivial case it suffices to define a field name and a selector so the scraper knows where to look for the corresponding value. The quotes scraper is a good example for that:

```yml
fields:
  dynamic:
    - name: "quote"
      location:
        selector: ".quoteText"
```

**Key: `location`**

However, it might be a bit more complex to extract the desired information. Take for instance the concert scraper configuration shown above, more specifically the config snippet for the `title` field.

```yml
fields:
  dynamic:
    - name: "title"
      location:
        selector: "h3"
        regex_extract:
          exp: "[^•]*"
          index: 0
```

This field is implicitly of type `text`. Other types, such as `url` or `date` would have to be configured with the keyword `type`. The `location` tells the scraper where to look for the field value and how to extract it. In this case the selector on its own would not be enough to extract the desired value as we would get something like this: `Bastian Baker • Konzert`. That's why there is an extra option to define a regular expression to extract a substring. Note that in this example our extracted string would still contain a trainling space which is automatically removed by the scraper. Let's have a look at two more examples to have a better understanding of the location configuration. Let's say we want to extract "Tonhalle-Orchester Zürich" from the following html snippet.

```html
<div class="member">
  <span class="member-name"></span>
  <span class="member-name"> Tonhalle-Orchester Zürich</span
  ><span class="member-function">, </span>
  <span class="member-name"> Yi-Chen Lin</span
  ><span class="member-function"> Leitung und Konzept,</span>
  <span class="composer"> Der Feuervogel </span>
  <span class="veranstalter"> Organizer: Tonhalle-Gesellschaft Zürich AG </span>
</div>
```

We can do this by configuring the location like this:

```yml
location:
  selector: ".member .member-name"
  node_index: 1 # This indicates that we want the second node (indexing starts at 0)
```

Last but not least let's say we want to extract the time "20h00" from the following html snippet.

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

Here, the selector is not enough to extract the desired string and we can't go further down the tree by using different selectors. With the `child_index` we can point to the exact string we want. A `child_index` of 0 would point to the first `<strong>` node, a `child_index` of 1 would point to the string containing "19h00", a `child_index` of 2 would point to the second `<strong>` node and finally a `child_index` of 3 points to the correct string. If `child_index` is set to -1 the first child that results in a regex match will be used. This can be usefull if the `child_index` varies across different items. In the current example however, the `child_index` is always the same but the string still contains more stuff than we need which is why we use a regular expression to extract the desired substring.

To get an even better feeling for the location configuration check out the numerous examples in the `concerts-config.yml` file.

**Key: `can_be_empty`**

This key only applies to dynamic fields of type text. As the name suggests, if set to `true` there won't be an error message if the value is empty.

**Key: `hide`**

This key determines whether a field should be exlcuded from the resulting item. This can be handy when you want to filter based on a field that you don't want to include in the actual item. For more information on filters checkout the **Filters** section below.

**Key: `on_subpage`**

This key indicates that the corresponding field value should be extracted from a subpage defined in another dynamic field of type `url`. In the following example the comment field will be extracted from the subpage who's url is the value of the dynamic field with the name "url".

```yml
dynamic:
  - name: "comment"
    location:
      selector: ".qt-the-content div"
    can_be_empty: true
    on_subpage: "url"
  - name: "url"
    type: "url"
    location:
      selector: ".qt-text-shadow"
```

**Key: `type`**

A dynamic field has a field type that can either be `text`, `url` or `date`. The default is `text`. In that case the string defined by the `location` is extracted and used 'as is' as the value for the respective field. The other types are:

- `url`

  A url has one additional boolean option: `relative`. This option determines whether this scraper's base url will be prepended to the string that has been extracted with the given `location`

- `date`

  A date field is different from a text field in that the result is a complete, valid date. Internally, this is a `time.Time` object but in the json output it is represented by a string. In order to be able to handle a lot of different cases where date information might be spread across different locations, might be formatted in different ways using different languages a date field has a list of components where each component looks like this:

  ```yml
  components:
    - covers:
        day: bool # optional
        month: bool # optional
        year: bool # optional
        time: bool # optional
      location:
        selector: "<selector>"
        ... # the location has the same configuration as explained above.
      layout: ["<layout>"]
  date_location: "Europe/Berlin"
  date_language: "it_IT"
  ```

  As can be seen, a component has to define which part of the date it covers (at least one part has to be covered). Next, the location of this component has to be defined. This is done the same way as we defined the location for a text field string. Finally, we need to define a list of possible layouts where each layout is defined the 'go-way' as this scraper is written in go. For more details check out [this](https://yourbasic.org/golang/format-parse-string-time-date-example/) link or have a look at the numerous examples in the `concerts-config.yml` file. Note that a layout string is always in English although the date string on the scraped website might be in a different language. Also note that mostly the layout list only contains one element. Only in rare cases where different events on the same site have different layouts it is necessary to define more than one layout.
  
  The `date_language` key needs to correspond to the language on the website. Currently, the default is `de_DE`. Note, that this doesn't matter for dates that only contain numbers. `date_location` sets the time zone of the respective date.

### Filters

Filters can be used to define what items should make it into the resulting list of items. A filter configuration looks as follows:

```yml
filters:
  - field: "status"
    regex: "cancelled"
    match: false
  - field: "status"
    regex: "delayed"
    match: false
```

The `field` key determines to which field the regular expression will be applied. `regex` defines the regular expression and `match` determines whether the item should be included or excluded on match. Note, that as soon as there is one match for a regular expression that has `match` set to **false** the respective item will be exlcuded from the results without looking at the other filters.

## Build & release

To build and release a new version of goskyr [Goreleaser](https://goreleaser.com/) is used, also see [Quick Start](https://goreleaser.com/quick-start/).

1. Run a "local-only" release to see if it works using the release command:

  ```bash
  goreleaser release --snapshot --rm-dist
  ```

2. Export github token

  ```bash
  export GITHUB_TOKEN="YOUR_GH_TOKEN"
  ```

3. Create a tag and push it to GitHub

  ```bash
  git tag -a v0.1.5 -m "new features"
  git push origin v0.1.5
  ```

4. Run GoReleaser at the root of this repository:

  ```bash
  goreleaser release
  ```

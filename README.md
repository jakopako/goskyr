# Event crawler

This project is meant to provide an easy way of extracting event data from websites. Ideally, one would
simply provide a URL pointing to the respective website and the crawler would do the rest. However, since this is not trivial, as an intermediate step towards this (unreachable?) goal on top of the URL a few other configuration parameters have to be defined in order for the crawler to find the respective event data.

The crawler is still in a very early stage but a few websites can already be crawled by simply adding an additional config snippet and without modifying the code.

It might be debatable whether go is the best language to implement such a crawler but I simply wanted to improve my go skills and at the same time implement this project idea I had.

## Event structure

Currently an event has the following fields:

```json
{
    "title": "ESMERALDA GALDA",
    "location": "Helsinki",
    "city": "Zurich",
    "date": "2022-01-13T19:00:00Z",
    "url": "https://www.helsinkiklub.ch",
    "comment": "die familie galda hautnah zur gala!",
    "type": "concert"
}
```

## Configuration

Have a look at the configuration file `config.yml` for details about how to configure the crawler for a specific website.

## Run the crawler

The crawler can be executed with `go run main.go` to crawl all configured locations and print the results. To run a single crawler add the flag `-single`. To write the events to the event api add the environment variables `API_USER`, `API_PASSWORD` and `EVENT_API` and add the flag `-store` to the go command.

## Regular execution through Github Actions

The crawler is regularely being executed through Github Actions and its crawled data consequently written to the event api described below.

## Event API

An API that provides basic functionality to query and manage event data has been implemented [here](https://github.com/jakopako/event-api) and is currently running [here](https://event-api-6bbi2ttrza-oa.a.run.app/). Checkout the [swagger doc](https://event-api-6bbi2ttrza-oa.a.run.app/api/swagger/index.html) to find out more.

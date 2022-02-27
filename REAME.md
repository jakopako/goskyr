# Go crawler

This project's goal is to make it easier to extract structured data from web pages. Initially, the main use case was to extract event data from
different venue websites. However, the code has been rewritten to handle the more general use case of extracting a list of items from any website.
This could be a list of books from an online book store, a list of plays in a public theater, a list of newspaper articles, etc. Currently, information can only be extracted from static websites.

TODO:

* motivation
* links to similar projects

## Configuration

Checkout the `example-config.yml` for details about how to configure the crawler. Basically, an extracted item can have static fields that are the same for each item and dynamic fields whose values are extracted from the respective website based on the given configuration.

### Static fields

### Dynamic fields

#### Field types

* `text`
* `url`
* `date`

JSTemplate
==========

Golang templates with embedded Javascript, using Duktape.

Example
-------

```go
package main

import (
  "log"
  "time"

  "github.com/cardigann/go-jstemplate"
)

func main() {
  t := jstemplate.New(`
    fetchCheerio('http://duktape.org/')
      .then(function($) {
        return $("#front-blurp p:first-of-type").html();
      });
    `)

  log.Println(t.Render())
  // Duktape is an <b>embeddable Javascript</b> engine, with a focus on <b>portability</b> and compact <b>footprint</b>.
}
```

package jstemplate

import (
	"fmt"
	"log"
	"net/http"

	cheerio "github.com/cardigann/go-duktape-cheerio"
	fetch "github.com/cardigann/go-duktape-fetch"
	duktape "gopkg.in/olebedev/go-duktape.v3"
)

var jsIncludes = `
	var Promise = Promise || fetch.Promise;

	function fetchResponseToText(fetchResponse) {
		return fetchResponse.text();
	}

	function fetchCheerio(url, options) {
		return fetch(url).then(fetchResponseToText).then(function(body){
			return cheerio.load(body, options);
		});
	}
`

var jsResolveTpl = `
	Promise.resolve(tpl)
		.then(function(body) {
			__success(body);
		})
		.catch(function(error) {
			print(error.stack)
			__error(error.message, error.lineNumber);
		});
`

type Template struct {
	src string
	rt  http.RoundTripper
}

func New(js string) *Template {
	return &Template{
		src: js,
	}
}

func (tpl *Template) RoundTrip(r *http.Request) (*http.Response, error) {
	log.Printf("Fetching %s via golang", r.URL.String())
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Printf("Response failed: %v", err)
		return nil, err
	}
	log.Printf("Response was %d", resp.StatusCode)
	return resp, err
}

func (tpl *Template) createContext() (*duktape.Context, error) {
	ctx := duktape.New()

	cheerio.Define(ctx)
	fetch.DefineWithRoundTripper(ctx, tpl)

	if err := ctx.PevalString(jsIncludes); err != nil {
		return nil, err
	}

	ctx.Pop()
	return ctx, nil
}

func (tpl *Template) Render() (string, error) {
	ctx, err := tpl.createContext()
	if err != nil {
		return "", err
	}

	defer ctx.Destroy()

	success := make(chan string)
	errs := make(chan error)

	ctx.PushGlobalGoFunction("__success", func(_ *duktape.Context) int {
		success <- ctx.SafeToString(-1)
		return 0
	})

	ctx.PushGlobalGoFunction("__error", func(_ *duktape.Context) int {
		line := ctx.GetInt(-1)
		message := ctx.SafeToString(-2)
		errs <- fmt.Errorf("Javascript error on line %d: %s", line, message)
		return 0
	})

	ctx.PushGlobalObject()

	if err := ctx.PevalString(tpl.src); err != nil {
		return "", err
	}

	ctx.PutPropString(-2, "tpl")
	ctx.Pop()

	if err := ctx.PevalString(jsResolveTpl); err != nil {
		return "", err
	}

	select {
	case result := <-success:
		return result, nil
	case err := <-errs:
		return "", err
	}
}

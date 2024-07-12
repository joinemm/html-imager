package main

import "fmt"

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"net/http"
)

// TODO: use config file
var port int = 8080
var addr string = fmt.Sprint("http://localhost:", port)
var staticDir string = "static"

// Main entrypoint
func main() {
	var browser *rod.Browser

	// Find chromium executable, panic if not found
	path, exists := launcher.LookPath()
	if !exists {
		panic("No chromium executable found!")
	}

	// Launch the headless browser instance
	u := launcher.New().Bin(path).MustLaunch()
	browser = rod.New().ControlURL(u).MustConnect()

	// Serve static files using http fileserver under /static/
	fs := http.FileServer(http.Dir(staticDir))
	http.Handle("/static/", http.StripPrefix("/static", fs))

	// Route that takes screenshot of some html string
	http.HandleFunc("/html", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Got a request for /html")

		// Get html content and css selector from Form data
		html := r.PostFormValue("html")
		selector := r.PostFormValue("selector")

		// Render the html in browser
		page := render_page(browser, html)

		// Get a big enough viewport that
		// our target element should be completely visible
		page.MustSetViewport(2048, 2048, 1.0, false)

		// Take screenshot of the page
		buf := get_screenshot(page, selector)

		// Return the image as the http response
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(buf)
	})

	// Run the webserver on specified port
	fmt.Println("Running on port", port)
	err := http.ListenAndServe(fmt.Sprint(":", port), nil)
	if err != nil {
		panic(err)
	}
}

// Takes screenshot of specified element on the given page
func get_screenshot(page *rod.Page, selector string) []byte {
	fmt.Println("Taking screenshot of:", selector)
	element := page.MustElement(selector)

	// Take screenshot in jpeg format and quality of 90
	buf, err := element.Screenshot(proto.PageCaptureScreenshotFormatJpeg, 90)
	if err != nil {
		panic(err)
	}

	// Remember to close the page as we don't need it anymore
	page.MustClose()
	return buf
}

// Visit some url and return the page
func go_to_url(browser *rod.Browser, url string) *rod.Page {
	fmt.Println("Visiting url: ", url)
	page := browser.MustPage(url)

	return page
}

// Get a new page with the given html string as it's content
func render_page(browser *rod.Browser, html string) *rod.Page {
	fmt.Println("Rendering page: ", html)
	// Set this webserver as the root url of the page,
	// so static content can be used in html with relative path
	page := browser.MustPage(addr)
	// Set the raw html as the content of the page
	page.MustSetDocumentContent(html)

	return page
}

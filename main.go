package main

import (
	"encoding/json"
	"fmt"
	"github.com/aymerick/raymond"
	"github.com/caarlos0/env/v11"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gorilla/handlers"
	"github.com/ysmood/gson"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var conf struct {
	Port        int    `env:"HOST" envDefault:"3000"`
	Host        string `env:"PORT" envDefault:"0.0.0.0"`
	StaticDir   string `env:"STATIC_DIR" envDefault:"static"`
	TemplateDir string `env:"TEMPLATE_DIR" envDefault:"templates"`
}

var browser *rod.Browser
var templates map[string]*raymond.Template

// Main entrypoint
func main() {
	parseConfig()
	parseTemplates()
	initBrowser()

	// Serve static files using http fileserver under /static/
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir(conf.StaticDir))))
	// API routes
	http.HandleFunc("/html", get_html)
	http.HandleFunc("/url", get_url)
	http.HandleFunc("/template", get_template)

	// Run the webserver on specified port
	addr := fmt.Sprint(conf.Host, ":", conf.Port)
	fmt.Println("Running on", addr)

	err := http.ListenAndServe(addr, handlers.LoggingHandler(os.Stdout, http.DefaultServeMux))
	if err != nil {
		panic(err)
	}
}

func initBrowser() {
	// Find chromium executable, panic if not found
	// This prevents the rod library from trying to download a precompiled chromium binary...
	path, exists := launcher.LookPath()
	if !exists {
		panic("No chromium executable found!")
	}
	// Launch the headless browser instance
	u := launcher.New().Bin(path).MustLaunch()
	browser = rod.New().ControlURL(u).MustConnect()
}

func parseConfig() {
	err := env.Parse(&conf)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", conf)
}

func parseTemplates() {
	templates = make(map[string]*raymond.Template)

	templateFiles, err := os.ReadDir(conf.TemplateDir)
	if err != nil {
		panic(err)
	}

	for _, e := range templateFiles {
		b, err := os.ReadFile(conf.TemplateDir + "/" + e.Name())
		if err != nil {
			panic(err)
		}
		source := string(b)
		tpl, err := raymond.Parse(source)
		if err != nil {
			panic(err)
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		templates[name] = tpl
		println("Parsed template:", name)
	}
}

func get_html(w http.ResponseWriter, r *http.Request) {
	html := r.URL.Query().Get("html")
	selector := r.URL.Query().Get("selector")

	page := html_page(browser, html)
	return_screenshot(w, page, selector)
}

func get_template(w http.ResponseWriter, r *http.Request) {
	template := r.URL.Query().Get("template")
	selector := r.URL.Query().Get("selector")

	var context map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&context)
	if err != nil {
		println("Error decoding json")
		panic(err)
	}

	context["STATIC"] = fmt.Sprint("http://127.0.0.1:", conf.Port, "/static")

	// Evaluate the template with supplied context
	result, err := templates[template].Exec(context)
	if err != nil {
		println("Error evaluating template")
		panic(err)
	}

	page := html_page(browser, result)
	return_screenshot(w, page, selector)
}

func get_url(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	selector := r.URL.Query().Get("selector")

	page := browser.MustPage(url)
	return_screenshot(w, page, selector)
}

// Takes screenshot of specified element on the given page
func take_screenshot(page *rod.Page, selector string) []byte {
	page.MustSetViewport(1, 1, 1.0, false)
	page.MustWaitLoad()

	var buf []byte
	var err error
	if selector != "" {
		element := page.MustElement(selector)
		buf, err = element.Screenshot(proto.PageCaptureScreenshotFormatJpeg, 90)
	} else {
		buf, err = page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormatJpeg,
			Quality: gson.Int(90),
		})
	}
	if err != nil {
		println("Error taking screenshot")
		panic(err)
	}

	// Close the page as we don't need it anymore
	page.MustClose()

	return buf
}

func return_screenshot(w http.ResponseWriter, page *rod.Page, selector string) {
	buf := take_screenshot(page, selector)

	// Return the image as the http response
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(buf)
}

// Get a new page with the given html string as it's content
func html_page(browser *rod.Browser, html string) *rod.Page {
	// Set this webserver as the root url of the page,
	// so static content can be used in html with relative path
	addr := fmt.Sprint("http://127.0.0.1", ":", conf.Port)
	page := browser.MustPage(addr)

	// Set the raw html as the content of the page
	page.MustSetDocumentContent(html)

	return page
}

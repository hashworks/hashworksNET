package acception

import (
	"flag"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/sclevine/agouti"
)

var (
	hub            string
	url            string
	screenshotPath string
	htmlPath       string
	page           *agouti.Page
)

func TestMain(m *testing.M) {
	flag.StringVar(&hub, "hub", "127.0.0.1:4444", "The hub address")
	flag.StringVar(&url, "url", "https://hashworks.net/", "The url to check")
	flag.StringVar(&screenshotPath, "screenshotPath", "", "Path to the screenshot file generated on failure, tmp file will be used otherwise")
	flag.StringVar(&htmlPath, "htmlPath", "", "Path to the html file generated on failure, tmp file will be used otherwise")
	flag.Parse

	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	fmt.Printf("Testing %s with hub on %s\n", url, hub)

	retCode := m.Run()

	if page != nil {
		if retCode != 0 { // Failed
			const tmpPrefix = "hashworksNET-Test-"

			// Try to create a screenshot
			var screenshotFile *os.File
			var err error
			if screenshotPath == "" {
				screenshotFile, err = ioutil.TempFile(os.TempDir(), tmpPrefix+"*.png")
			} else {
				screenshotFile, err = os.Create(screenshotPath)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, "Failed to create screenshot file: "+err.Error())
			} else {
				defer screenshotFile.Close()
				err := page.Screenshot(screenshotFile.Name())

				if err != nil {
					fmt.Fprintln(os.Stderr, "Failed to create screenshot: "+err.Error())
				} else {
					fmt.Fprintln(os.Stderr, "Created a screenshot at "+screenshotFile.Name())
				}
			}

			//Try to save html
			var htmlFile *os.File
			if htmlPath == "" {
				htmlFile, err = ioutil.TempFile(os.TempDir(), tmpPrefix+"*.html")
			} else {
				htmlFile, err = os.Create(htmlPath)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, "Failed to create html file: "+err.Error())
			} else {
				defer htmlFile.Close()
				html, err := page.HTML()

				if err != nil {
					fmt.Fprintln(os.Stderr, "Failed to fetch html: "+err.Error())
				} else {
					bytes, err := htmlFile.WriteString(html)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Failed to save html: "+err.Error())
					} else {
						fmt.Fprintf(os.Stderr, "Wrote %d bytes of HTML to %s\n", bytes, htmlFile.Name())
					}
				}
			}
		}
		page.Destroy()
	}

	os.Exit(retCode)
}

func TestAcception(t *testing.T) {
	var err error

	// Connect to hub
	// Note: We are not destroying the page at the end of this function since we are doing that in TestMain
	page, err = agouti.NewPage(fmt.Sprintf("http://%s/wd/hub", hub), agouti.Browser("chrome"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// MAIN CHECK
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	// Open main page
	err = page.Navigate(url)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// Header should contain some links
	header := page.FindByXPath("//header")
	for _, link := range []string{"/", "/status", "https://git.hashworks.net", "https://github.com/hashworks", "https://steamcommunity.com/id/hashworks", "https://www.reddit.com/user/hashworks/posts/"} {
		linkElement := header.FindByXPath(fmt.Sprintf("//nav//a[@href='%s']", link))
		if count, err := linkElement.Count(); assert.NoError(t, err) {
			assert.Equal(t, count, 1)
		}
	}
	if t.Failed() {
		t.FailNow()
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// CONTACT PAGE CHECK
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	// Click contact, should navigate to current page
	if !assert.NoError(t, page.FindByLink("contact").Click()) {
		t.FailNow()
	}
	currentURL, err := page.URL()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, currentURL, url)

	// Check title
	title, err := page.Title()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, title, "/home/hashworks")

	// Check article
	article := page.FindByXPath("//article[@class='card full']")

	articleHeader, err := article.FindByXPath("//h1").Text()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, articleHeader, "Contact")

	// Should contain some links
	for _, link := range []string{"mailto:mail@hashworks.net", "/.well-known/openpgpkey/hu/dizb37aqa5h4skgu7jf1xjr4q71w4paq", "https://libera.chat/"} {
		linkElement := article.FindByXPath(fmt.Sprintf("//a[@href='%s']", link))
		if count, err := linkElement.Count(); assert.NoError(t, err) {
			assert.Equal(t, count, 1)
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// STATUS PAGE CHECK
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	var statusBackgroundImages []string

	// Click status, should navigate to /status and title should be /status as well
	if !assert.NoError(t, page.FindByLink("status").Click()) {
		t.FailNow()
	}
	currentURL, err = page.URL()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, currentURL, fmt.Sprintf("%sstatus", url))

	title, err = page.Title()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, title, "/home/hashworks/status")

	// Check article

	articles := page.AllByXPath("//article[contains(@class, 'card')]")
	articleCount, err := articles.Count()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, 2, articleCount)

	for i := 0; i < articleCount; i++ {
		article := articles.At(i)

		articleHeader, err = article.FindByXPath("h1").Text()
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		articleTag, err := article.FindByXPath("div[@class='tag']").Text()
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		if i == 1 {
			assert.Equal(t, "Public Services", articleHeader)
			assert.Equal(t, "HIVE", articleTag)

			articleSubHeaders := article.AllByXPath("h4")
			articleSubHeadersCount, err := articleSubHeaders.Count()
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.Equal(t, 3, articleSubHeadersCount)

			for i := 0; i < articleSubHeadersCount; i++ {
				articleSubHeader := articleSubHeaders.At(i)
				articleSubHeaderText, err := articleSubHeader.Text()
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				if i == 0 {
					assert.Equal(t, "Plex:", articleSubHeaderText)
				} else if i == 1 {
					assert.Equal(t, "ZNC:", articleSubHeaderText)
				} else if i == 2 {
					assert.Equal(t, "Upstream Load:", articleSubHeaderText)
				}
			}

			serviceStatusDivs := article.AllByXPath("div[contains(@class, 'status')]")
			serviceStatusDivsCount, err := serviceStatusDivs.Count()
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.Equal(t, 3, serviceStatusDivsCount)
		} else if i == 0 {
			class := "load"
			assert.Equal(t, articleHeader, "Server Load")
			assert.Equal(t, "HIVE", articleTag)

			backgroundImage, err := article.FindByXPath(fmt.Sprintf("div[@class='status-svg']/div[@class='%s']", class)).CSS("background-image")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.True(t, strings.HasPrefix(backgroundImage, `url("`)) {
				t.FailNow()
			}

			statusBackgroundImages = append(statusBackgroundImages, strings.Split(backgroundImage, `"`)[1])
		} else {
			t.FailNow()
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// STATUS IMAGE CHECK
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	// Check status graphs
	for _, link := range statusBackgroundImages {
		resp, err := http.Get(link)
		if assert.NoError(t, err) {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			svg := string(body)

			// Should be an SVG
			assert.True(t, strings.HasPrefix(svg, "<svg"))

			// Should contain some paths
			pathRegex := regexp.MustCompile("<path")
			matches := pathRegex.FindAllStringIndex(svg, -1)
			assert.True(t, len(matches) >= 10)
		}
	}
}

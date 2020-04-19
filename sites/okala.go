package sites

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"strings"
	"superMarketsDownloader/models"
	"superMarketsDownloader/utils"
	"time"
)

func parseTitle(dom *goquery.Selection) string {
	return dom.Find(".h4.line-height-sm.font-weight-bold").First().Text()
}

func parseDescription(dom *goquery.Selection) string {
	return dom.Find(".subtitle2.text-muted").First().Text()
}

func parseOldPrice(dom *goquery.Selection) string {
	oldPriceElement := dom.Find("description-list.description-list-horizontal.description-list-horizontal-sm.mb-0 .text-muted del")
	var oldPrice string
	if oldPriceElement.Size() > 0 {
		oldPrice = oldPriceElement.First().Text()
	}
	return oldPrice
}


func parsePrice(dom *goquery.Selection) string {
	priceElement := dom.Find(".description-list.description-list-horizontal.description-list-horizontal-sm.mb-0 .text-primary span")
	var price string
	if priceElement.Size() > 0 {
		price = priceElement.Text()
	}
	return price
}


func parseContent(dom *goquery.Selection) (content string, err error) {
	return dom.Find("#description").First().Html()
}


func parseAttributes(dom *goquery.Selection) [][]string {
	attributesElement := dom.Find(".description-list.description-list-horizontal").First()
	var attributes [][]string
	attribute := make([]string, 2)
	attributesElement.Children().Each(func(i int, selection *goquery.Selection) {
		if i % 2 == 0 {
			attribute[0], _ = selection.Html()
		}
		if i % 2 == 1 {
			attribute[1], _ = selection.Html()
			attributes = append(attributes, attribute)
			attribute = make([]string, 2)
		}
	})
	return attributes
}


func parseBrandFromAttributes(attributes *[][]string) string {
	for _, tuple := range *attributes {
		if strings.Contains(tuple[0], "برند") {
			if len(tuple) > 1 {
				return tuple[1]
			}
		}
	}
	return ""
}


func parseImages(dom *goquery.Selection) *[]models.ProductImage {
	imagesElement := dom.Find(".gallery-top img")
	var images []models.ProductImage
	imagesElement.Each(func(i int, selection *goquery.Selection) {
		imageUrl, _ := selection.Attr("data-zoom-image")
		if strings.HasPrefix(imageUrl, "/") {
			imageUrl = "https://okala.com" + imageUrl
		}
		images = append(images, models.ProductImage{Url:imageUrl})
	})
	return &images
}


func parseCategory(dom *goquery.Selection) *[]string {
	var categories []string
	dom.Find(".breadcrumb").Children().Each(func(i int, selection *goquery.Selection) {
		categories = append(categories, selection.Find("a").Text())
	})
	if len(categories) > 0 {
		categories = categories[:len(categories) - 1]
	}
	return &categories
}


func parseProductId(url string) string {
	return strings.ReplaceAll(utils.GetPathFromUrl(url), "/", "")
}

func ParseOkalaProduct(url string) *models.Product {
	c := colly.NewCollector(colly.AllowedDomains(utils.GetHostFromUrl(url)))
	var product *models.Product = &models.Product{}
	c.OnHTML("html", func(element *colly.HTMLElement) {
		dom := element.DOM
		title := parseTitle(dom)
		description := parseDescription(dom)
		oldPrice := parseOldPrice(dom)
		price := parsePrice(dom)
		content, _ :=  parseContent(dom)
		attributes := parseAttributes(dom)
		attributesJson, _ := json.Marshal(attributes)
		brand := parseBrandFromAttributes(&attributes)
		images := *parseImages(dom)
		categories := *parseCategory(dom)
		pid := parseProductId(url)
		product = &models.Product{
			Url:              url,
			Images:           images,
			Price:            price,
			Description:      description,
			JsonMeta:         string(attributesJson),
			Brand:            brand,
			ShortDescription: "",
			Title:            title,
			Category:         categories,
			ParsedAt:         time.Now(),
			Content:          content,
			OldPrice:         oldPrice,
			Pid:              pid,
		}
	})
	c.Visit(url)
	c.Wait()
	return product
}


type (
	OkalaParser struct {
		websiteInfo *models.WebsiteInfo
	}
)


func (parser *OkalaParser) Info() *models.WebsiteInfo {
	return parser.websiteInfo
}


func (parser *OkalaParser) DownloadProduct(url string) *models.Product {
	return ParseOkalaProduct(url)
}

func (parser *OkalaParser) IsValidProduct(url string) bool {
	return utils.IsNumber(strings.ReplaceAll(utils.GetPathFromUrl(url), "/", ""))
}

func GetOkalaParser() Parser {
	return &OkalaParser{
		websiteInfo: &models.WebsiteInfo{
			WebsiteUrl:"okala.com",
			SiteMapIndexUrl:"",
			SiteMaps:[]models.SiteMap{{Url:"https://okala.com/sitemap.xml"}},
		},
	}
}
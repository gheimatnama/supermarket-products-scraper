package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// It must be able to resume latest download
var uid *int64
var downloadPath *string
var website *string
var workersCount *int

func GetProductValidator(websiteUrl string) func(string) bool {
	switch websiteUrl {
	case "okala.com":
		return func(url string) bool {
			return isNumber(strings.ReplaceAll(GetPathFromUrl(url), "/", ""))
		}
	case "snapp.market":
		return func(url string) bool {
			return strings.Contains(url, "products")
		}
	}
	return func(url string) bool {
		return false
	}
}


func GetDownloadUid() *int64 {
	return flag.Int64("uid", time.Now().Unix(), "Uid to make your download able to be resumed later")
}


func GetWebsite() *string {
	return flag.String("website", "okala.com", "Website to be downloaded")
}

func GetWorkersCount() *int {
	return flag.Int("workers", 10, "Total workers to scrape pages and images")
}

func GetDownloadPath() *string {
	return flag.String("path", "websites", "Path for downloads")
}


func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
	return true, err
}


func GetUidPath() string {
	return *downloadPath + "/" + strconv.FormatInt(*uid, 10)
}

func GetCachedWebsiteInfo(websiteInfo *WebsiteInfo) bool {
	websiteInfoPath := GetUidPath() + "/info-" + websiteInfo.WebsiteUrl + ".json"
	if fileExists, err := exists(websiteInfoPath); fileExists && err == nil {
		file, err := os.Open(websiteInfoPath)
		if err != nil {
			panic(err)
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		json.Unmarshal(content, websiteInfo)
		file.Close()
		return true
	}
	return false
}

func FillWebSiteInfo(websiteInfo *WebsiteInfo) *WebsiteInfo {
	exists := GetCachedWebsiteInfo(websiteInfo)
	if exists {
		return websiteInfo
	}
	var siteMaps []SiteMap
	if websiteInfo.SiteMapIndexUrl == "" {
		siteMaps = websiteInfo.SiteMaps
	} else {
		siteMaps = ParseSiteMapIndex(websiteInfo.SiteMapIndexUrl)
	}
	for i, _ := range siteMaps {
		ParseSiteMap(&siteMaps[i], GetProductValidator(websiteInfo.WebsiteUrl))
	}
	websiteInfo.SiteMaps = siteMaps
	PersistWebSiteInfo(websiteInfo)
	return websiteInfo
}


func PersistWebSiteInfo(websiteInfo *WebsiteInfo) {
	websiteInfoPath := GetUidPath() + "/info-" + websiteInfo.WebsiteUrl + ".json"
	content, err := json.Marshal(websiteInfo)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(websiteInfoPath, content, os.ModePerm)
	if err != nil {
		panic(err)
	}
}


func ParseSiteMap(siteMap *SiteMap, isValidUrl func(string) bool) {
	var knownUrls []string
	c := colly.NewCollector(colly.AllowedDomains(GetHostFromUrl(siteMap.Url)))
	c.OnXML("//urlset/url/loc", func(e *colly.XMLElement) {
		if isValidUrl(e.Text) {
			knownUrls = append(knownUrls, e.Text)
		}
	})
	c.Visit(siteMap.Url)
	c.Wait()
	siteMap.SiteMapsUrl = knownUrls
}

func GetWebsiteInfo(website string) *WebsiteInfo {
	var websiteInfo WebsiteInfo
	switch website {
	case "snapp.market":
		websiteInfo = WebsiteInfo{WebsiteUrl:"snapp.market", SiteMapIndexUrl:"https://core.snapp.market/sitemap.xml"}
	case "okala.com":
		websiteInfo = WebsiteInfo{WebsiteUrl:"okala.com", SiteMapIndexUrl:"", SiteMaps:[]SiteMap{SiteMap{Url:"https://okala.com/sitemap.xml"}}}
	}
	return FillWebSiteInfo(&websiteInfo)
}


type (
	WebsiteInfo struct {
		WebsiteUrl string
		SiteMaps []SiteMap
		SiteMapIndexUrl string
		CurrentSiteMapUrlPosition int
	}
	SiteMap struct {
		Url string
		LocalFilePath string
		SiteMapsUrl []string
		Products []*Product
	}

	ProductImage struct {
		Url       string
		LocalPath string
		ParsedAt  time.Time
	}

	Product struct {
		Url string
		Images []ProductImage
		Price string
		Description string
		JsonMeta string
		Brand string
		ShortDescription string
		Title string
		Category []string
		ParsedAt time.Time
		Content string
		OldPrice string
		Pid string
	}
)

func isNumber(str string) bool {
	if _, err := strconv.Atoi(str); err == nil {
		return true
	}
	return false
}


func StringToNumber(str string) int {
	if number, err := strconv.Atoi(str); err == nil {
		return number
	}
	return 0
}


func HashString(str string) string {
	md5HashInBytes := md5.Sum([]byte("Sum returns bytes"))
	md5HashInString := hex.EncodeToString(md5HashInBytes[:])
	return md5HashInString
}


func (image *ProductImage) DownloadImage(info *WebsiteInfo, product *Product, wg *sync.WaitGroup) {
	_, err := url.ParseRequestURI(image.Url)
	if err != nil {
		wg.Done()
		return
	}
	resp, err := http.Get(image.Url)
	if err != nil {
		wg.Done()
		return
	}
	contentType := resp.Header.Get(http.CanonicalHeaderKey("content-type"))
	fileExtensions, err := mime.ExtensionsByType(contentType)
	if err != nil {
		fileExtensions = []string{".jpg"}
	}
	path := GetUidPath() + "/" + info.WebsiteUrl + "/" + product.Pid + "/"
	os.MkdirAll(path, os.ModePerm)
	localPath := path + HashString(image.Url) + fileExtensions[0]
	file, err := os.Create(localPath)
	if err != nil {
		panic(err)
	}
	io.Copy(file, resp.Body)
	file.Close()
	resp.Body.Close()
	image.LocalPath = localPath
	image.ParsedAt = time.Now()
	wg.Done()
}


func ParseOkalaProduct(url string) *Product {
	c := colly.NewCollector(colly.AllowedDomains(GetHostFromUrl(url)))
	var product *Product = &Product{}
	c.OnHTML("html", func(element *colly.HTMLElement) {
		dom := element.DOM
		title := dom.Find(".h4.line-height-sm.font-weight-bold").First().Text()
		description := dom.Find(".subtitle2.text-muted").First().Text()
		oldPriceElement := dom.Find("description-list.description-list-horizontal.description-list-horizontal-sm.mb-0 .text-muted del")
		var oldPrice string
		if oldPriceElement.Size() > 0 {
			oldPrice = oldPriceElement.First().Text()
		}
		priceElement := dom.Find(".description-list.description-list-horizontal.description-list-horizontal-sm.mb-0 .text-primary span")
		var price string
		if priceElement.Size() > 0 {
			price = priceElement.Text()
		}
		content, _ := dom.Find("#description").First().Html()
		attributesElement := dom.Find(".description-list.description-list-horizontal").First()
		var attributes [][]string
		attribute := make([]string, 2)
		var brand string
		attributesElement.Children().Each(func(i int, selection *goquery.Selection) {
			if i % 2 == 0 {
				attribute[0], _ = selection.Html()
				if strings.Contains(selection.Text(), "برند") {
					brand = selection.Text()
				}
			}
			if i % 2 == 1 {
				attribute[1], _ = selection.Html()
				attributes = append(attributes, attribute)
				attribute = make([]string, 2)
			}
		})
		attributesJson, _ := json.Marshal(attributes)
		imagesElement := dom.Find(".gallery-top img")
		var images []ProductImage
		imagesElement.Each(func(i int, selection *goquery.Selection) {
			imageUrl, _ := selection.Attr("data-zoom-image")
			if strings.HasPrefix(imageUrl, "/") {
				imageUrl = "https://okala.com" + imageUrl
			}
			images = append(images, ProductImage{Url:imageUrl})
		})
		var categories []string
		dom.Find(".breadcrumb").Children().Each(func(i int, selection *goquery.Selection) {
			categories = append(categories, selection.Find("a").Text())
		})
		if len(categories) > 0 {
			categories = categories[:len(categories) - 1]
		}
		pid := strings.ReplaceAll(GetPathFromUrl(url), "/", "")
		product = &Product{
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



var myClient = &http.Client{Timeout: 10 * time.Second}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, target)
	if err != nil {
		return err
	}
	return nil
}


func ParseSnappProduct(productUrl string) *Product {
	pathParts := strings.Split(strings.ReplaceAll(productUrl, "https://snapp.market/", ""), "/")
	id := pathParts[2]
	type (
		SnappProductImage struct {
			Image string `json:"image"`
			Thumb string `json:"thumb"`
		}
		SnappProduct struct {
			Id int `json:"id"`
			Title string `json:"title"`
			Description string `json:"description"`
			Content string `json:"content"`
			OldPrice int `json:"price"`
			DiscountedPrice int `json:"discounted_price"`
			Images []SnappProductImage `json:"images"`
			Brand string `json:"brand"`
			HtmlDescription string `json:"html_description"`
			MetaDescription string `json:"meta_description"`
		}

		SnappCategory struct {
			Id int `json:"id"`
			Title string `json:"title"`
			Slug string `json:"slug"`
		}

		SnappJson struct {
			Product SnappProduct `json:"product"`
			Breadcrumb []SnappCategory `json:"breadcrumb"`
		}

	)
	result := &SnappJson{}
	err := getJson("https://core.snapp.market/api/v1/vendors/0r5ryz/products/" + id, result)
	if err != nil {
		fmt.Println(err)
		return &Product{}
	}
	var images []ProductImage
	for _, image := range result.Product.Images {
		images = append(images, ProductImage{Url: image.Image})
	}
	var categories []string
	for _, category := range result.Breadcrumb {
		categories = append(categories, category.Title)
	}
	return &Product{
		Url:              productUrl,
		Images:           images,
		Price:            strconv.Itoa(result.Product.DiscountedPrice),
		Description:      result.Product.Description,
		JsonMeta:         "{}",
		Brand:            result.Product.Brand,
		ShortDescription: result.Product.HtmlDescription,
		Title:            result.Product.Title,
		Category:         categories,
		ParsedAt:         time.Now(),
		Content:          result.Product.Content,
		OldPrice:         strconv.Itoa(result.Product.OldPrice),
		Pid:              strconv.Itoa(result.Product.Id),
	}
}



func DownloadProduct(url string, info *WebsiteInfo, productChan chan *Product, wg *sync.WaitGroup, limit chan bool) {
	defer wg.Done()
	var product *Product
	switch info.WebsiteUrl {
	case "okala.com":
		product = ParseOkalaProduct(url)
	case "snapp.market":
		product = ParseSnappProduct(url)
	}
	if product.Images != nil {
		DownloadImages(product, info)
	}
	<-limit
	productChan <- product
}


func DownloadImages(product *Product, info *WebsiteInfo) {
	var wg sync.WaitGroup
	wg.Add(len(product.Images))
	for i, _ := range product.Images {
		go product.Images[i].DownloadImage(info, product, &wg)
	}
	wg.Wait()
}


func WriteDownloadedProducts(productsChan chan *Product, siteMap *SiteMap, info *WebsiteInfo, wg *sync.WaitGroup) {
	total := info.CurrentSiteMapUrlPosition
	defer wg.Done()
	for product := range productsChan {
		total++
		fmt.Println("Total downloaded => " + strconv.Itoa(total) + " , Sitemap links => " + strconv.Itoa(len(siteMap.SiteMapsUrl)) + " , Current sitemap downloaded products => " + strconv.Itoa(len(siteMap.Products)))
		if product.Title == "" {
			continue
		}
		siteMap.Products = append(siteMap.Products, product)
		info.CurrentSiteMapUrlPosition++
		if total % 50 == 0 {
			PersistWebSiteInfo(info)
		}
	}
	PersistWebSiteInfo(info)
}


func DownloadProducts(info *WebsiteInfo) {
	var wg sync.WaitGroup
	var productsChanWg sync.WaitGroup
	for i := 0; i < len(info.SiteMaps); i++  {
		siteMap := &info.SiteMaps[i]
		if len(siteMap.SiteMapsUrl) == 0 {
			continue
		}
		productsChan := make(chan *Product, 10)
		limit := make(chan bool, *workersCount)
		productsChanWg.Add(1)
		go WriteDownloadedProducts(productsChan, siteMap, info, &productsChanWg)
		wg.Add(len(siteMap.SiteMapsUrl))
		for j := info.CurrentSiteMapUrlPosition; j < len(siteMap.SiteMapsUrl); j++ {
			url := siteMap.SiteMapsUrl[j]
			limit <- true
			go DownloadProduct(url, info, productsChan, &wg, limit)
		}
		wg.Wait()
		close(productsChan)
		productsChanWg.Wait()
		info.CurrentSiteMapUrlPosition = 0
		PersistWebSiteInfo(info)
	}
}


func GetHostFromUrl(path string) string {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return u.Hostname()
}


func GetPathFromUrl(path string) string {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return u.Path
}


func ParseSiteMapIndex(siteMapIndexUrl string) []SiteMap {
	var knownSiteMaps []SiteMap
	c := colly.NewCollector(colly.AllowedDomains(GetHostFromUrl(siteMapIndexUrl)))
	c.OnXML("//sitemapindex/sitemap/loc", func(e *colly.XMLElement) {
		knownSiteMaps = append(knownSiteMaps, SiteMap{Url:e.Text})
	})
	c.Visit(siteMapIndexUrl)
	c.Wait()
	return knownSiteMaps
}


func PrepareFileSystem() {
	os.MkdirAll(*downloadPath, os.ModePerm)
	os.Mkdir(*downloadPath + "/" + strconv.FormatInt(*uid, 10), os.ModePerm)
}


func ShowInfo() {
	fmt.Println("Scraping website ", *website, " with ", *workersCount, " workers. Output folder : ", *downloadPath)
}

func main() {
	downloadPath = GetDownloadPath()
	uid = GetDownloadUid()
	website = GetWebsite()
	workersCount = GetWorkersCount()
	flag.Parse()
	PrepareFileSystem()
	ShowInfo()
	info := GetWebsiteInfo(*website)
	DownloadProducts(info)
}
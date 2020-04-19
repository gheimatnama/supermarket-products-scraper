package sites

import (
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"superMarketsDownloader/models"
	"superMarketsDownloader/utils"
	"sync"
	"time"
)


type Parser interface {
	Info() *models.WebsiteInfo
	DownloadProduct(string) *models.Product
	IsValidProduct(string) bool
}


func ParseSiteMap(siteMap *models.SiteMap, isValidUrl func(string) bool) {
	var knownUrls []string
	c := colly.NewCollector(colly.AllowedDomains(utils.GetHostFromUrl(siteMap.Url)))
	c.OnXML("//urlset/url/loc", func(e *colly.XMLElement) {
		if isValidUrl(e.Text) {
			knownUrls = append(knownUrls, e.Text)
		}
	})
	c.Visit(siteMap.Url)
	c.Wait()
	siteMap.SiteMapsUrl = knownUrls
}


func ParseSiteMapIndex(siteMapIndexUrl string) []models.SiteMap {
	var knownSiteMaps []models.SiteMap
	c := colly.NewCollector(colly.AllowedDomains(utils.GetHostFromUrl(siteMapIndexUrl)))
	c.OnXML("//sitemapindex/sitemap/loc", func(e *colly.XMLElement) {
		knownSiteMaps = append(knownSiteMaps, models.SiteMap{Url:e.Text})
	})
	c.Visit(siteMapIndexUrl)
	c.Wait()
	return knownSiteMaps
}

func getUidPath(info *models.RuntimeInfo) string {
	return info.DownloadPath + "/" + strconv.FormatInt(info.Uid, 10)
}


func DownloadImage(image *models.ProductImage, info *models.WebsiteInfo, runtimeInfo *models.RuntimeInfo,  product *models.Product, wg *sync.WaitGroup) {
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
	path := getUidPath(runtimeInfo) + "/" + info.WebsiteUrl + "/" + product.Pid + "/"
	os.MkdirAll(path, os.ModePerm)
	localPath := path + utils.HashString(image.Url) + fileExtensions[0]
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


func GetCachedWebsiteInfo(websiteInfo *models.WebsiteInfo, info *models.RuntimeInfo) bool {
	websiteInfoPath := getUidPath(info) + "/info-" + websiteInfo.WebsiteUrl + ".json"
	if fileExists, err := utils.Exists(websiteInfoPath); fileExists && err == nil {
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



func FillWebSiteInfo(parser Parser, info *models.RuntimeInfo) *models.WebsiteInfo {
	websiteInfo := parser.Info()
	exists := GetCachedWebsiteInfo(websiteInfo, info)
	if exists {
		return websiteInfo
	}
	var siteMaps []models.SiteMap
	if websiteInfo.SiteMapIndexUrl == "" {
		siteMaps = websiteInfo.SiteMaps
	} else {
		siteMaps = ParseSiteMapIndex(websiteInfo.SiteMapIndexUrl)
	}
	for i, _ := range siteMaps {
		ParseSiteMap(&siteMaps[i], parser.IsValidProduct)
	}
	websiteInfo.SiteMaps = siteMaps
	PersistWebSiteInfo(websiteInfo, info)
	return websiteInfo
}


func PersistWebSiteInfo(websiteInfo *models.WebsiteInfo, info *models.RuntimeInfo) {
	websiteInfoPath := getUidPath(info) + "/info-" + websiteInfo.WebsiteUrl + ".json"
	content, err := json.Marshal(websiteInfo)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(websiteInfoPath, content, os.ModePerm)
	if err != nil {
		panic(err)
	}
}



func DownloadProduct(url string, parser Parser, runtimeInfo *models.RuntimeInfo, productChan chan *models.Product, wg *sync.WaitGroup, limit chan bool) {
	defer wg.Done()
	var product *models.Product
	product = parser.DownloadProduct(url)
	if product.Images != nil {
		DownloadImages(product, parser.Info(), runtimeInfo)
	}
	<-limit
	productChan <- product
}


func DownloadImages(product *models.Product, info *models.WebsiteInfo, runtimeInfo *models.RuntimeInfo) {
	var wg sync.WaitGroup
	wg.Add(len(product.Images))
	for i, _ := range product.Images {
		go DownloadImage(&product.Images[i], info, runtimeInfo, product, &wg)
	}
	wg.Wait()
}


func WriteDownloadedProducts(productsChan chan *models.Product, siteMap *models.SiteMap, info *models.WebsiteInfo, runtimeInfo *models.RuntimeInfo, wg *sync.WaitGroup) {
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
			PersistWebSiteInfo(info, runtimeInfo)
		}
	}
	PersistWebSiteInfo(info, runtimeInfo)
}
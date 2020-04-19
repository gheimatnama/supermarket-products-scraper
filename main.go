package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"superMarketsDownloader/models"
	"superMarketsDownloader/sites"
	"sync"
	"time"
)


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



func DownloadProducts(parser sites.Parser, runtimeInfo *models.RuntimeInfo) {
	var wg sync.WaitGroup
	info := parser.Info()
	var productsChanWg sync.WaitGroup
	for i := 0; i < len(info.SiteMaps); i++  {
		siteMap := &info.SiteMaps[i]
		if len(siteMap.SiteMapsUrl) == 0 {
			continue
		}
		productsChan := make(chan *models.Product, 10)
		limit := make(chan bool, runtimeInfo.WorkersCount)
		productsChanWg.Add(1)
		go sites.WriteDownloadedProducts(productsChan, siteMap, info, runtimeInfo, &productsChanWg)
		wg.Add(len(siteMap.SiteMapsUrl))
		for j := info.CurrentSiteMapUrlPosition; j < len(siteMap.SiteMapsUrl); j++ {
			url := siteMap.SiteMapsUrl[j]
			limit <- true
			go sites.DownloadProduct(url, parser, runtimeInfo, productsChan, &wg, limit)
		}
		wg.Wait()
		close(productsChan)
		productsChanWg.Wait()
		info.CurrentSiteMapUrlPosition = 0
		sites.PersistWebSiteInfo(info, runtimeInfo)
	}
}


func ShowInfo(info *models.RuntimeInfo) {
	fmt.Println("Scraping website ", info.Website, " with ", info.WorkersCount, " workers. Output folder : ", info.DownloadPath)
}


func GetParsers() map[string]sites.Parser{
	websites := make(map[string]sites.Parser)
	websites["okala.com"] = sites.GetOkalaParser()
	websites["snapp.market"] = sites.GetSnapParser()
	return websites
}


func GetRuntimeInfo() *models.RuntimeInfo {
	downloadPath := GetDownloadPath()
	uid := GetDownloadUid()
	website := GetWebsite()
	workersCount := GetWorkersCount()
	flag.Parse()
	return &models.RuntimeInfo{
		DownloadPath: *downloadPath,
		Uid:          *uid,
		Website:      *website,
		WorkersCount: *workersCount,
	}
}


func PrepareRuntime(runtimeInfo *models.RuntimeInfo) {
	os.MkdirAll(runtimeInfo.DownloadPath, os.ModePerm)
	os.Mkdir(runtimeInfo.DownloadPath + "/" + strconv.FormatInt(runtimeInfo.Uid, 10), os.ModePerm)
}

func GetWebsiteParser(info *models.RuntimeInfo) sites.Parser {
	parsers := GetParsers()
	parser := parsers[info.Website]
	sites.FillWebSiteInfo(parser.(sites.Parser), info)
	return parser.(sites.Parser)
}

func main() {
	runtimeInfo := GetRuntimeInfo()
	PrepareRuntime(runtimeInfo)
	ShowInfo(runtimeInfo)
	DownloadProducts(GetWebsiteParser(runtimeInfo), runtimeInfo)
}
package models

import "time"

type (

	RuntimeInfo struct {
		DownloadPath string
		Uid int64
		Website string
		WorkersCount int
	}

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

package sites

import (
	"fmt"
	"strconv"
	"strings"
	"superMarketsDownloader/models"
	"superMarketsDownloader/utils"
	"time"
)

type (
	snappProductImage struct {
		Image string `json:"image"`
		Thumb string `json:"thumb"`
	}

	snappProduct struct {
		Id int `json:"id"`
		Title string `json:"title"`
		Description string `json:"description"`
		Content string `json:"content"`
		OldPrice int `json:"price"`
		DiscountedPrice int `json:"discounted_price"`
		Images []snappProductImage `json:"images"`
		Brand string `json:"brand"`
		HtmlDescription string `json:"html_description"`
		MetaDescription string `json:"meta_description"`
	}

	snappCategory struct {
		Id int `json:"id"`
		Title string `json:"title"`
		Slug string `json:"slug"`
	}

	snappJson struct {
		Product snappProduct `json:"product"`
		Breadcrumb []snappCategory `json:"breadcrumb"`
	}
)

func ParseSnappProduct(productUrl string) *models.Product {
	pathParts := strings.Split(strings.ReplaceAll(productUrl, "https://snapp.market/", ""), "/")
	id := pathParts[2]
	result := &snappJson{}
	err := utils.GetJson("https://core.snapp.market/api/v1/vendors/0r5ryz/products/" + id, result)
	if err != nil {
		fmt.Println(err)
		return &models.Product{}
	}
	var images []models.ProductImage
	for _, image := range result.Product.Images {
		images = append(images, models.ProductImage{Url: image.Image})
	}
	var categories []string
	for _, category := range result.Breadcrumb {
		categories = append(categories, category.Title)
	}
	return &models.Product{
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


type (
	SnapParser struct {
		websiteInfo *models.WebsiteInfo
	}
)


func (parser *SnapParser) Info() *models.WebsiteInfo {
	return parser.websiteInfo
}


func (parser *SnapParser) DownloadProduct(url string) *models.Product {
	return ParseSnappProduct(url)
}

func (parser *SnapParser) IsValidProduct(url string) bool {
	return utils.IsNumber(strings.ReplaceAll(utils.GetPathFromUrl(url), "/", ""))
}

func GetSnapParser() Parser{
	return &SnapParser{
		websiteInfo: &models.WebsiteInfo{
			WebsiteUrl:"snapp.market",
			SiteMapIndexUrl:"https://core.snapp.market/sitemap.xml",
		},
	}
}

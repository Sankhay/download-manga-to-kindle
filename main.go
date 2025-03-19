package main

import (
	"context"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Sankhay/download-manga/configs"
	"github.com/Sankhay/download-manga/utils"
	"github.com/chromedp/chromedp"
	"github.com/go-shiori/go-epub"
	"github.com/nfnt/resize"
	"golang.org/x/image/webp"
)

func main() {

	var url string

	fmt.Println("Link do capitulo: ")
	fmt.Println("Exemplo: https://slimeread.com/ler/429/cap-01")
	fmt.Scanln(&url)

	mangaChapter := utils.ExtractLastNumbers(url)

	imagesLinks, titleName, err := getImgsLinksAndTitle(url)

	if err != nil {
		log.Fatal(err)
	}

	if len(imagesLinks) >= 2 {
		imagesLinks = imagesLinks[:len(imagesLinks)-2]
	} else {
		imagesLinks = nil
	}

	imagesPath, err := downloadImages(imagesLinks)

	defer utils.DeleteAllImages()

	if err != nil {
		log.Printf("Failed to download image: %v", err)
	}

	imagesPathUpdated, err := treatImages(imagesPath)

	if err != nil {
		log.Fatal(`error treating image: %w`, err)
	}

	epubFileName := fmt.Sprintf(`%v_%v.epub`, *titleName, mangaChapter)

	err = createEpubfile(epubFileName, imagesPathUpdated)

	if err != nil {
		log.Printf(`failed to create e pub file: %v`, err)
	}

	fmt.Printf(`epub file created with success: %v`, epubFileName)
}

func getImgsLinksAndTitle(url string) ([]string, *string, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
	)

	var imagesLinks []string

	initialXtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)

	defer cancel()

	ctx, cancel := chromedp.NewContext(initialXtx)
	defer cancel()

	var titleName string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`document.readyState === 'complete'`, nil),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('div.flex.tw-as.tw-rl.tw-vp.tw-dea img')).map(img => img.src)`, &imagesLinks),
		chromedp.Text(`a.tw-xc.tw-qa.tw-lk.link.tw-nl`, &titleName, chromedp.ByQuery),
	)

	if err != nil {
		return nil, nil, fmt.Errorf(`error getting imgs links and title: %w`, err)
	}

	return imagesLinks, &titleName, nil
}

func downloadImages(imagesLinks []string) ([]string, error) {

	var imagesPath []string

	if err := os.MkdirAll(configs.ImageDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create images directory: %w", err)
	}

	for i, imgLink := range imagesLinks {
		resp, err := http.Get(imgLink)

		if err != nil {
			return nil, fmt.Errorf("failed to download image: %w", err)
		}

		ext := filepath.Ext(imgLink)

		fileName := fmt.Sprintf("downloaded_image_%d%v", i, ext)

		filePath := filepath.Join(configs.ImageDir, fileName)

		defer resp.Body.Close()

		outFile, err := os.Create(filePath)

		if err != nil {
			return nil, fmt.Errorf("failed to create file %w", err)
		}

		defer outFile.Close()

		_, err = io.Copy(outFile, resp.Body)

		if err != nil {
			return nil, fmt.Errorf("failed to save image: %w", err)
		}

		imagesPath = append(imagesPath, filePath)

	}

	return imagesPath, nil
}

func treatImages(imagesPath []string) ([]string, error) {

	var imagesPathUpdated []string

	for _, imgPath := range imagesPath {
		imgExt := filepath.Ext(imgPath)

		switch imgExt {
		case ".png":
			imgPath, err := convertPNGtoJPG(imgPath)

			if err != nil {
				return nil, fmt.Errorf(`error converting img to jpg: %w`, err)
			}

			imagesPathUpdated = append(imagesPathUpdated, *imgPath)
		case ".webp":
			imgPath, err := convertWEBPtoJPG(imgPath)

			if err != nil {
				return nil, fmt.Errorf(`error converting img to jpg: %w`, err)
			}

			imagesPathUpdated = append(imagesPathUpdated, *imgPath)
		default:
			imagesPathUpdated = append(imagesPathUpdated, imgPath)
		}
	}

	return imagesPathUpdated, nil
}

func convertPNGtoJPG(inputPath string) (*string, error) {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PNG file: %v", err)
	}
	defer inFile.Close()

	img, err := png.Decode(inFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %v", err)
	}

	outFilePath := utils.ChangeExtensionToJpg(inputPath)

	outFile, err := os.Create(outFilePath)

	if err != nil {
		return nil, fmt.Errorf("failed to create JPG file: %v", err)
	}

	defer outFile.Close()
	options := &jpeg.Options{Quality: configs.ImagesQuality}

	err = jpeg.Encode(outFile, img, options)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPG: %v", err)
	}

	return &outFilePath, nil
}

func convertWEBPtoJPG(inputPath string) (*string, error) {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open WebP file: %v", err)
	}
	defer inFile.Close()

	img, err := webp.Decode(inFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode WebP: %v", err)
	}

	resizedImg := resize.Resize(600, 800, img, resize.Lanczos3)

	outFilePath := utils.ChangeExtensionToJpg(inputPath)

	outFile, err := os.Create(outFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create JPG file: %v", err)
	}
	defer outFile.Close()

	// Set JPEG quality options (0-100)
	options := &jpeg.Options{Quality: configs.ImagesQuality}

	// Encode image to JPG format
	err = jpeg.Encode(outFile, resizedImg, options)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPG: %v", err)
	}

	return &outFilePath, nil
}

func createEpubfile(epubFileName string, mangaImgsPath []string) error {
	e, err := epub.NewEpub(epubFileName)

	if err != nil {
		return fmt.Errorf(`failed to create epub: %w`, err)
	}

	var allContent string

	cssPath, err := e.AddCSS("style.css", "")

	if err != nil {
		return fmt.Errorf(`failed to add css file to epub: %w`, err)
	}

	for _, imgPath := range mangaImgsPath {

		imgPathInEpub, err := e.AddImage(imgPath, "")

		if err != nil {
			return fmt.Errorf(`failed adding image to epub: %w`, err)
		}

		content := fmt.Sprintf(`<img src="%s" />`, imgPathInEpub)

		allContent += content
	}

	_, err = e.AddSection(allContent, "Section 1", "section1.xhtml", cssPath)

	if err != nil {
		return fmt.Errorf(`failed to add section to epub: %w`, err)
	}

	err = e.Write(filepath.Join(configs.EpubDir, epubFileName))

	if err != nil {
		return fmt.Errorf(`failed to create epub file: %w`, err)
	}

	return nil
}

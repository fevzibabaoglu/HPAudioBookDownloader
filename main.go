package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

type Chapter struct {
	Title            string `json:"title"`
	Subtitle         string `json:"subtitle"`
	Audio            string `json:"audio"`
	BuyURL           string `json:"buyUrl"`
	DownloadURL      string `json:"downloadUrl"`
	DownloadFilename string `json:"downloadFilename"`
	Cover            string `json:"cover"`
	Lyrics           string `json:"lyrics"`
}

const (
	TempFileFormat = "chapter_%d.mp3"
	ConcatFileName = "concat_list.txt"
	FfmpegLogLevel = "error"
)

func requestChapterURLs(bookURL string) ([]string, error) {
	resp, err := http.Get(bookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch book URL: %v", err)
	}
	defer resp.Body.Close()

	var chapters []Chapter
	if err := json.NewDecoder(resp.Body).Decode(&chapters); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	var chapterURLs []string
	for _, chapter := range chapters {
		if chapter.Audio != "" {
			chapterURLs = append(chapterURLs, chapter.Audio)
		}
	}
	return chapterURLs, nil
}

func downloadChapter(chapterURL string) ([]byte, error) {
	resp, err := http.Get(chapterURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download chapter: %v", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func saveToFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}

func createConcatFile(tempFiles []string) error {
	f, err := os.Create(ConcatFileName)
	if err != nil {
		return fmt.Errorf("failed to create concat file: %v", err)
	}
	defer f.Close()

	for _, tempFile := range tempFiles {
		if _, err := f.WriteString(fmt.Sprintf("file '%s'\n", tempFile)); err != nil {
			return fmt.Errorf("failed to write to concat file: %v", err)
		}
	}
	return nil
}

func combineAudioFiles(outputFilePath string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel", FfmpegLogLevel,
		"-f", "concat",
		"-safe", "0",
		"-i", ConcatFileName,
		"-c", "copy",
		outputFilePath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %v", err)
	}
	return nil
}

func cleanUp(files []string) {
	for _, file := range files {
		_ = os.Remove(file)
	}
}

func downloadAndCombineBook(bookURL, outputFilePath string) error {
	chapterURLs, err := requestChapterURLs(bookURL)
	if err != nil {
		return fmt.Errorf("error fetching chapter URLs: %v", err)
	}

	tempFiles := []string{}
	for i, chapterURL := range chapterURLs {
		fmt.Printf("Downloading chapter %d from %s\n", i+1, chapterURL)
		chapterData, err := downloadChapter(chapterURL)
		if err != nil {
			return fmt.Errorf("error downloading chapter %d: %v", i+1, err)
		}

		tempFile := fmt.Sprintf(TempFileFormat, i+1)
		if err := saveToFile(tempFile, chapterData); err != nil {
			return fmt.Errorf("error saving chapter %d: %v", i+1, err)
		}
		tempFiles = append(tempFiles, tempFile)
	}

	if err := createConcatFile(tempFiles); err != nil {
		return err
	}

	if err := combineAudioFiles(outputFilePath); err != nil {
		return err
	}

	cleanUp(append(tempFiles, ConcatFileName))

	fmt.Printf("Book successfully saved to %s\n", outputFilePath)
	return nil
}

func main() {
	const basePath = `YOUR_OUTPUT_BASE_PATH`
	books := []struct {
		BookURL        string
		OutputFilename string
	}{
		{"https://hpaudiobook.online/?audioigniter_playlist_id=639", "harry-potter-and-philosophers-stone.mp3"},
		{"https://hpaudiobook.online/?audioigniter_playlist_id=647", "harry-potter-and-chamber-of-secrets.mp3"},
		{"https://hpaudiobook.online/?audioigniter_playlist_id=652", "harry-potter-and-prisoner-of-azkaban.mp3"},
		{"https://hpaudiobook.online/?audioigniter_playlist_id=656", "harry-potter-and-the-goblet-of-fire.mp3"},
		{"https://hpaudiobook.online/?audioigniter_playlist_id=660", "harry-potter-and-the-order-of-the-phoenix.mp3"},
		{"https://hpaudiobook.online/?audioigniter_playlist_id=664", "harry-potter-and-the-half-blood-prince.mp3"},
		{"https://hpaudiobook.online/?audioigniter_playlist_id=668", "harry-potter-and-the-deathly-hallows.mp3"},
	}

	for _, book := range books {
		outputFilePath := fmt.Sprintf("%s/%s", basePath, book.OutputFilename)
		if err := downloadAndCombineBook(book.BookURL, outputFilePath); err != nil {
			fmt.Printf("Error processing book '%s': %v\n", book.OutputFilename, err)
		} else {
			fmt.Printf("Successfully processed '%s'.\n", book.OutputFilename)
		}
	}
}

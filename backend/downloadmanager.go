package backend

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

type DownloadStatus int

const (
	DownloadStatusNotStarted DownloadStatus = iota
	DownloadStatusDownloading
	DownloadStatusCompleted
	DownloadStatusError
)

type DownloadManager struct {
}

type Download struct {
	status DownloadStatus

	onStatusChange func(*Download)
}

func (d *Download) Status() DownloadStatus { return d.status }

func (d *Download) OnStatusChange(cb func(*Download)) {
	d.onStatusChange = cb
}

func (d *Download) setStatus(s DownloadStatus) {
	d.status = s
	if d.onStatusChange != nil {
		d.onStatusChange(d)
	}
}

func (dm *DownloadManager) DownloadTrack(
	downloadName string,
	filePath string,
	provider mediaprovider.MediaProvider,
	track *mediaprovider.Track,
) *Download {
	d := &Download{status: DownloadStatusNotStarted}
	go func() {
		reader, err := provider.DownloadTrack(track.ID)
		if err != nil {
			log.Println(err)
			d.setStatus(DownloadStatusError)
			return
		}

		file, err := os.Create(filePath)
		if err != nil {
			log.Println(err)
			d.setStatus(DownloadStatusError)
			return
		}
		defer file.Close()

		_, err = io.Copy(file, reader)
		if err != nil {
			log.Println(err)
			d.setStatus(DownloadStatusError)
			return
		}
		d.setStatus(DownloadStatusCompleted)
	}()
	return d
}

func (dm *DownloadManager) DownloadTracks(
	downloadName string,
	filePath string,
	provider mediaprovider.MediaProvider,
	tracks []*mediaprovider.Track,
) *Download {
	d := &Download{status: DownloadStatusNotStarted}
	go func() {
		zipFile, err := os.Create(filePath)
		if err != nil {
			log.Println(err)
			d.setStatus(DownloadStatusError)
		}
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()

		for _, track := range tracks {
			reader, err := provider.DownloadTrack(track.ID)
			if err != nil {
				log.Println(err)
				continue
			}

			fileName := filepath.Base(track.FilePath)

			fileWriter, err := zipWriter.Create(fileName)
			if err != nil {
				log.Println(err)
				continue
			}

			_, err = io.Copy(fileWriter, reader)
			if err != nil {
				log.Println(err)
				continue
			}

			log.Printf("Saved track %s to: %s\n", track.Name, filePath)
		}
		d.setStatus(DownloadStatusCompleted)
	}()
	return d
}

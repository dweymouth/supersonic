package backend

import (
	"archive/zip"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/util"
)

type DownloadManager struct {
}

// DownloadStatus represents the status of a download
type DownloadStatus int

const (
	DownloadStatusNotStarted DownloadStatus = iota
	DownloadStatusDownloading
	DownloadStatusCompleted
	DownloadStatusCanceled
	DownloadStatusError
)

// Download represents an individual download task.
type Download struct {
	status DownloadStatus

	ctx    context.Context
	cancel context.CancelFunc

	onStatusChange func(*Download)
}

// Status returns the download status.
func (d *Download) Status() DownloadStatus { return d.status }

// OnStatusChange sets a callback that will be invoked when the download status changes.
func (d *Download) OnStatusChange(cb func(*Download)) {
	d.onStatusChange = cb
}

func (d *Download) Cancel() {
	d.cancel()
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
	d.ctx, d.cancel = context.WithCancel(context.Background())
	go func(d *Download) {
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

		d.setStatus(DownloadStatusDownloading)
		reader = util.NewCancellableReader(d.ctx, reader)
		_, err = io.Copy(file, reader)
		if d.ctx.Err() == context.Canceled {
			os.Remove(filePath)
			d.setStatus(DownloadStatusCanceled)
			return
		}
		if err != nil {
			log.Println(err)
			d.setStatus(DownloadStatusError)
			return
		}
		d.setStatus(DownloadStatusCompleted)
	}(d)
	return d
}

func (dm *DownloadManager) DownloadTracks(
	downloadName string,
	filePath string,
	provider mediaprovider.MediaProvider,
	tracks []*mediaprovider.Track,
) *Download {
	d := &Download{status: DownloadStatusNotStarted}
	go func(d *Download) {
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

			reader = util.NewCancellableReader(d.ctx, reader)
			_, err = io.Copy(fileWriter, reader)
			if d.ctx.Err() == context.Canceled {
				os.Remove(filePath)
				d.setStatus(DownloadStatusCanceled)
				return
			}
			if err != nil {
				log.Println(err)
				continue
			}

			log.Printf("Saved track %s to: %s\n", track.Name, filePath)
		}
		d.setStatus(DownloadStatusCompleted)
	}(d)
	return d
}

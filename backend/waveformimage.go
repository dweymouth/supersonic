package backend

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/util"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/supersonic-app/go-mpv"
)

type WaveformImageGenerator struct {
	audioCache *AudioCache
}

type WaveformImage = image.NRGBA

func NewWaveformImage() *WaveformImage {
	return image.NewNRGBA(image.Rect(0, 0, 1024, 32))
}

type WaveformImageJob struct {
	ItemID   string
	lock     sync.Mutex
	img      *WaveformImage
	err      error
	progress int // first invalid pixel in X direction
	cancel   func()
}

func (w *WaveformImageJob) Cancel() {
	if w != nil && w.cancel != nil {
		w.cancel()
	}
}

func (w *WaveformImageJob) Done() bool {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.err != nil || w.progress >= w.img.Bounds().Dx()
}

func (w *WaveformImageJob) Err() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.err
}

func (w *WaveformImageJob) Get() *WaveformImage {
	if w.Done() {
		log.Println("returning image directly")
		return w.img
	}
	// return a new *WaveformImage with data copied
	// from the valid region of w.img
	height := w.img.Bounds().Dy()
	result := NewWaveformImage()

	// Copy each scanline from w.img to result
	for y := 0; y < height; y++ {
		srcOffset := w.img.PixOffset(0, y)
		dstOffset := result.PixOffset(0, y)
		copy(result.Pix[dstOffset:dstOffset+w.progress*4], w.img.Pix[srcOffset:srcOffset+w.progress*4])
	}

	return result
}

func NewWaveformImageGenerator(cache *AudioCache) *WaveformImageGenerator {
	return &WaveformImageGenerator{audioCache: cache}
}

func (w *WaveformImageGenerator) StartWaveformGeneration(item *mediaprovider.Track) *WaveformImageJob {
	ctx, cancel := context.WithCancel(w.audioCache.rootCtx)
	job := &WaveformImageJob{
		img:    NewWaveformImage(),
		ItemID: item.ID,
		cancel: cancel,
	}

	// Set up a pipeline of concurrent tasks that need to complete to generate
	// a waveform image:
	// 1. Begin downloading the file from the server
	// 2. Begin transcoding it to WAV
	// 3. Begin analyzing the resulting WAV file
	// 4. Begin generating the image from the analysis data
	go func() {
		path := w.audioCache.PathForCachedOrDownloadingFile(job.ItemID)
		// wait for file to begin downloading if not already
		for path == "" {
			time.Sleep(10 * time.Millisecond)
			if e := ctx.Err(); e != nil {
				job.setError(e)
				return
			}
			path = w.audioCache.PathForCachedOrDownloadingFile(job.ItemID)
		}

		dir := filepath.Dir(path)
		transcodeFile := filepath.Join(dir, filepath.Base(path)+"_waveform.wav")

		fileDone := func() bool {
			return w.audioCache.PathForCachedFile(job.ItemID) != ""
		}

		// If file isn't fully downloaded from server,
		// stream it to MPV via HTTP so it doesn't possibly
		// terminate the conversion to WAV early encountering EOF
		if !fileDone() {
			srv, err := util.NewFileStreamerServer(path, fileDone)
			if err != nil {
				job.setError(err)
				return
			}
			path = srv.Addr()
			log.Println("streaming file to MPV at ", path)
			go srv.Serve()
		}

		// Start converting the file to WAV for analysis
		var wavConvertDone bool
		go func() {
			err := convertToWav(ctx, path, transcodeFile)
			wavConvertDone = true
			if err != nil {
				job.setError(err)
			}
		}()

		// Wait for transcoded WAV file to begin being written
		for {
			if s, err := os.Stat(transcodeFile); err == nil && s.Size() > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
			if e := ctx.Err(); e != nil {
				job.setError(e)
				return
			}
		}

		// Start analyzing the converted wav file
		data := &waveformData{}
		go func() {
			err := analyzeWavFile(ctx, transcodeFile, data, item.Duration.Milliseconds(), func() bool { return wavConvertDone })
			if err != nil {
				log.Println("error analyzing wav", err.Error())
				job.setError(err)
			}
		}()

		// Start generating the waveform image
		go generateWaveformImage(ctx, data, job)
	}()
	return job
}

type waveformData struct {
	Peak [1024]byte
	RMS  [1024]byte

	progress int // first invalid index for Peak/RMS data
}

func generateWaveformImage(ctx context.Context, data *waveformData, job *WaveformImageJob) {
	centerY := job.img.Rect.Dy() / 2 // 16
	top := centerY - 1
	bottom := centerY

	opaqueColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	translucentColor := color.NRGBA{R: 255, G: 255, B: 255, A: 128}

	for x := 0; x < 1024; x++ {
		if data.progress <= x {
			time.Sleep(10 * time.Millisecond)
			if ctx.Err() != nil {
				return // expired
			}
		}

		rms := float64(data.RMS[x]) / 255.0
		peak := float64(data.Peak[x]) / 255.0

		rmsPixels := int(rms * 16)
		peakPixels := int((peak - rms) * 16)

		// Always draw at least 2 center pixels
		setPixel(job.img, x, top, opaqueColor)
		setPixel(job.img, x, bottom, opaqueColor)

		// Draw RMS pixels (solid)
		for i := 1; i <= rmsPixels; i++ {
			setPixel(job.img, x, top-i, opaqueColor)
			setPixel(job.img, x, bottom+i, opaqueColor)
		}

		// Draw Peak extension (translucent)
		for i := 1; i <= peakPixels; i++ {
			setPixel(job.img, x, top-rmsPixels-i, translucentColor)
			setPixel(job.img, x, bottom+rmsPixels+i, translucentColor)
		}
		job.progress = x + 1
	}
	log.Println("done generating image")
}

func analyzeWavFile(ctx context.Context, transcodeFile string, data *waveformData, millisecs int64, fileDone func() bool) error {
	if fileDone() {
		log.Println("Analyzing completely written file!!")
	}
	f, err := os.Open(transcodeFile)
	if err != nil {
		log.Println("error opening transcoded file")
		return err
	}
	defer f.Close()
	//defer os.Remove(transcodeFile)

	reader := trackingReader{rs: f}

	decoder := wav.NewDecoder(&reader)
	if !decoder.IsValidFile() {
		return errors.New("invalid wav file")
	}

	format := decoder.Format()

	totalSamples := format.SampleRate * int(millisecs) / 1000
	samplesPerChunk := totalSamples / 1024

	if err := decoder.FwdToPCM(); err != nil {
		return err
	}

	buf := &audio.IntBuffer{Data: make([]int, 4096)}
	curChunk := 0
	chunkSamples := make([]float32, 0, samplesPerChunk)

	// file read loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !fileDone() {
			// Check how many samples we can safely read without encountering EOF
			// and adjust read buffer size accordingly

			stat, err := f.Stat()
			if err != nil {
				return fmt.Errorf("stat failed: %w", err)
			}
			currentSize := stat.Size()

			readableBytes := currentSize - reader.Pos() // how many bytes are still available

			// Estimate how many samples we can read
			bytesPerSample := int64(2 * format.NumChannels) // 16-bit = 2 bytes per channel
			maxSamples := int(readableBytes / bytesPerSample)

			if maxSamples <= 0 {
				// Wait for more data to be written to file
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Resize buffer to fit only what’s safe
			safeSamples := maxSamples
			if safeSamples > cap(buf.Data) {
				safeSamples = cap(buf.Data)
			}
			buf.Data = buf.Data[:safeSamples]
		} else {
			// File is done being written, resize read buf to the max
			buf.Data = buf.Data[:cap(buf.Data)]
		}

		n, err := decoder.PCMBuffer(buf)
		if n == 0 || err == io.EOF {
			if fileDone() {
				data.progress = 1024 // set progress to done
			}
			if err == io.EOF && !fileDone() {
				return errors.New("WAV read got premature EOF")
			}
			break
		}
		if err != nil {
			return err
		}

		// Process samples
		for i := 0; i < n; i += format.NumChannels {
			sum := 0
			for c := 0; c < format.NumChannels; c++ {
				sum += buf.Data[i+c]
			}
			avg := float64(sum) / float64(format.NumChannels)
			// TODO: this assumes 16 bit
			sample := float32(avg / float64(1<<15)) // Normalize to [-1, 1]
			chunkSamples = append(chunkSamples, sample)

			if len(chunkSamples) >= samplesPerChunk {
				if curChunk < 1024 {
					peak, rms := computePeakAndRMS(chunkSamples)
					data.Peak[curChunk] = float32ToByte(peak)
					data.RMS[curChunk] = float32ToByte(rms)
				}
				curChunk++
				data.progress = curChunk
				chunkSamples = chunkSamples[:0]
				if curChunk >= 1024 {
					break
				}
			}
		}
	}

	// analyze the last chunk if it's partially filled with samples
	if curChunk < 1024 && len(chunkSamples) > 0 {
		peak, rms := computePeakAndRMS(chunkSamples)
		data.Peak[curChunk] = float32ToByte(peak)
		data.RMS[curChunk] = float32ToByte(rms)
		data.progress = curChunk + 1
	}

	log.Println("final chunk is", curChunk)

	return nil
}

func (j *WaveformImageJob) setError(err error) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.err = err
}

func computePeakAndRMS(chunk []float32) (peak float32, rms float32) {
	var sumSquares float64
	peak = 0.0
	for _, v := range chunk {
		abs := float32(math.Abs(float64(v)))
		if abs > peak {
			peak = abs
		}
		sumSquares += float64(v * v)
	}
	rms = float32(math.Sqrt(sumSquares / float64(len(chunk))))
	return
}

func float32ToByte(val float32) byte {
	if val > 1.0 {
		val = 1.0
	}
	if val < 0.0 {
		val = 0.0
	}
	return byte(val * 255)
}

func convertToWav(ctx context.Context, inPath, outPath string) error {
	m := mpv.Create()
	m.SetOptionString("video", "no")
	m.SetOptionString("audio-display", "no")
	m.SetOptionString("terminal", "no")
	m.SetOptionString("idle", "yes")
	m.SetOptionString("ao-pcm-file", outPath)
	m.SetOptionString("ao", "pcm")
	m.SetOption("volume", mpv.FORMAT_INT64, 100)
	// no need to preserve full sample resolution just for waveform image
	// let's make less data to process and smaller on-disk file
	m.SetOption("audio-samplerate", mpv.FORMAT_INT64, 22050)
	m.SetOptionString("audio-channels", "mono")
	m.SetOptionString("audio-format", "s16")
	if err := m.Initialize(); err != nil {
		return err
	}

	m.Command([]string{"loadfile", inPath, "replace"})

	// Wait for MPV idle or ctx expiry
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			ia := m.GetPropertyString("idle-active")
			if ia == "yes" || ia == "true" {
				return nil
			}
			// use small timeout to allow detecting ctx expiry
			// without too much delay
			e := m.WaitEvent(0.05 /*timeout seconds*/)
			if e.Event_Id == mpv.EVENT_IDLE {
				return nil
			}
		}
	}
}

func setPixel(img *image.NRGBA, x, y int, c color.NRGBA) {
	offset := img.PixOffset(x, y)
	img.Pix[offset+0] = c.R
	img.Pix[offset+1] = c.G
	img.Pix[offset+2] = c.B
	img.Pix[offset+3] = c.A
}

// wrap an io.ReadSeeker with support for tracking bytes read (Pos())
type trackingReader struct {
	rs  io.ReadSeeker
	pos int64
}

func (t *trackingReader) Read(p []byte) (int, error) {
	n, err := t.rs.Read(p)
	t.pos += int64(n)
	return n, err
}

func (t *trackingReader) Seek(offset int64, whence int) (int64, error) {
	newPos, err := t.rs.Seek(offset, whence)
	if err == nil {
		t.pos = newPos
	}
	return newPos, err
}

func (t *trackingReader) Pos() int64 {
	return t.pos
}

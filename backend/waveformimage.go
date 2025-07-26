package backend

import (
	"context"
	"errors"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/dweymouth/supersonic/backend/util"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/supersonic-app/go-mpv"
)

type WaveformData struct {
	Peak [1024]byte
	RMS  [1024]byte
}

type WaveformImage = image.NRGBA

func NewWaveformImage() *WaveformImage {
	return image.NewNRGBA(image.Rect(0, 0, 1024, 32))
}

func GenerateWaveformImage(data *WaveformData, imgbuf *WaveformImage, c color.Color) {
	centerY := imgbuf.Rect.Dy() / 2 // 16
	top := centerY - 1
	bottom := centerY

	// Convert the input color to RGBA
	r, g, b, _ := c.RGBA()
	opaqueColor := color.NRGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: 255,
	}
	translucentColor := color.NRGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: 128, // 50% opacity
	}

	for x := 0; x < 1024; x++ {
		rms := float64(data.RMS[x]) / 255.0
		peak := float64(data.Peak[x]) / 255.0

		rmsPixels := int(rms * 16)
		peakPixels := int((peak - rms) * 16)

		// Always draw at least 2 center pixels
		setPixel(imgbuf, x, top, opaqueColor)
		setPixel(imgbuf, x, bottom, opaqueColor)

		// Draw RMS pixels (solid)
		for i := 1; i <= rmsPixels; i++ {
			setPixel(imgbuf, x, top-i, opaqueColor)
			setPixel(imgbuf, x, bottom+i, opaqueColor)
		}

		// Draw Peak extension (translucent)
		for i := 1; i <= peakPixels; i++ {
			setPixel(imgbuf, x, top-rmsPixels-i, translucentColor)
			setPixel(imgbuf, x, bottom+rmsPixels+i, translucentColor)
		}
	}
}

func GetWaveformDataForFile(ctx context.Context, fpath string, fileIsDone func() bool) (*WaveformData, error) {
	dir := filepath.Dir(fpath)
	transcodeFile := filepath.Join(dir, filepath.Base(fpath)+"_waveform.wav")

	if !fileIsDone() {
		srv, err := util.NewFileStreamerServer(fpath, fileIsDone)
		if err != nil {
			return nil, err
		}
		fpath = srv.Addr()
		log.Println("streaming file to MPV at ", fpath)
		go srv.Serve()
	}

	err := convertToWav(ctx, fpath, transcodeFile)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(transcodeFile)
	if err != nil {
		log.Println("error opening transcoded file")
		return nil, err
	}
	defer f.Close()
	defer os.Remove(transcodeFile)

	decoder := wav.NewDecoder(f)
	if !decoder.IsValidFile() {
		return nil, errors.New("invalid wav file")
	}

	dur, err := decoder.Duration()
	if err != nil {
		return nil, err
	}

	format := decoder.Format()

	totalSamples := format.SampleRate * int(dur.Milliseconds()) / 1000
	samplesPerChunk := totalSamples / 1024

	if err := decoder.FwdToPCM(); err != nil {
		return nil, err
	}

	buf := &audio.IntBuffer{Data: make([]int, 4096)}
	data := &WaveformData{}
	curChunk := 0
	chunkSamples := make([]float32, 0, samplesPerChunk)
	for {
		n, err := decoder.PCMBuffer(buf)
		if n == 0 || err == io.EOF {
			break
		}
		if err != nil {
			return data, err
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
				chunkSamples = chunkSamples[:0]
				if curChunk >= 1024 {
					break
				}
			}
		}
	}

	// Optionally fill the last chunk if it's partially filled
	if curChunk < 1024 && len(chunkSamples) > 0 {
		peak, rms := computePeakAndRMS(chunkSamples)
		data.Peak[curChunk] = float32ToByte(peak)
		data.RMS[curChunk] = float32ToByte(rms)
	}

	return data, nil
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

	return mpvWaitForIdle(ctx, m)
}

func setPixel(img *image.NRGBA, x, y int, c color.NRGBA) {
	offset := img.PixOffset(x, y)
	img.Pix[offset+0] = c.R
	img.Pix[offset+1] = c.G
	img.Pix[offset+2] = c.B
	img.Pix[offset+3] = c.A
}

func mpvWaitForIdle(ctx context.Context, m *mpv.Mpv) error {
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
			e := m.WaitEvent(0.1 /*timeout seconds*/)
			if e.Event_Id == mpv.EVENT_IDLE {
				return nil
			}
		}
	}
}

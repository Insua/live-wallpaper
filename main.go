package main

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/util/grand"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var (
	cancelFunc context.CancelFunc
	mu         sync.Mutex
)

func main() {
	_, _ = gcron.AddSingleton(context.Background(), "# */30 * * * *", func(ctx context.Context) {
		fmt.Println(time.Now())
		randGif(false)
	})
	randGif(true)
	select {}
}

func randGif(isFirst bool) {
	tmpPath := "/tmp/live-wallpaper"
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)
	videos, _ := gfile.ScanDir(filepath.Join(exPath, "mp4"), "*.mp4")
	rand := grand.N(0, len(videos)-1)
	video := videos[rand]
	videoMd5 := gmd5.MustEncryptFile(video)
	tmpFilePath := filepath.Join(tmpPath, videoMd5)
	if !gfile.Exists(tmpFilePath) {
		_ = gfile.Mkdir(tmpFilePath)
	}
	pngs, _ := gfile.ScanDirFile(tmpFilePath, "*.png")
	if len(pngs) == 0 {
		if isFirst {
			animate(filepath.Join(exPath, "loading"))
		}
		convert(video, tmpFilePath)
		animate(tmpFilePath)
	} else {
		animate(tmpFilePath)
	}
}

func animate(dp string) {
	mu.Lock()
	if cancelFunc != nil {
		cancelFunc()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancelFunc = cancel
	mu.Unlock()

	fs, _ := gfile.ScanDirFile(dp, "*.png")
	amountOfFrames := len(fs)
	speed := time.Duration(float64(1) / float64(amountOfFrames) * 1e9)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for _, file := range fs {
					select {
					case <-ctx.Done():
						return
					default:
						runFeh(file)
						time.Sleep(speed)
					}
				}
			}
		}
	}()
}

func runFeh(fp string) {
	cmd := exec.Command("feh", "--bg-fill", "--no-fehbg", fp)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error running feh: %v\n", err)
	}
}

func convert(mp4Path, outPath string) {
	ffOut := outPath + "/%04d.png"
	cmd := exec.Command("ffmpeg", "-i", mp4Path, "-vf", "fps=10", "-s", "2560x1440", ffOut)
	_, _ = cmd.CombinedOutput()
}

package main

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/grand"
	"os/exec"
	"sync"
	"time"
)

var (
	cancelFunc context.CancelFunc
	mu         sync.Mutex
)

func main() {
	_, _ = gcron.AddSingleton(context.Background(), "0 0 * * * *", func(ctx context.Context) {
		randGif()
	})
	randGif()
	select {}
}

func randGif() {
	fp := gcmd.GetArg(1)
	dirs, _ := gfile.ScanDir(gconv.String(fp), "*")
	rand := grand.N(0, len(dirs)-1)
	dir := dirs[rand]
	animate(dir)
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

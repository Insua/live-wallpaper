package main

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/util/grand"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var (
	cancelFunc context.CancelFunc
	mu         sync.Mutex
)

func main() {
	addr := "/tmp/live-wallpaper/message.sock"

	if len(os.Args) > 1 {
		sendMessage(addr)
		return
	}

	if _, err := os.Stat(addr); err == nil {
		err = os.Remove(addr)
		if err != nil {
			panic(fmt.Sprintf("无法删除遗留的 socket 文件: %s, 错误: %v", addr, err))
		}
	}

	ln, err := net.Listen("unix", addr)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = ln.Close()
	}()

	go func() {
		for {
			_, err = ln.Accept()
			if err != nil {
				fmt.Println("连接错误:", err)
				continue
			}
			go randGif(false)
		}
	}()

	_, _ = gcron.AddSingleton(context.Background(), "# */30 * * * *", func(ctx context.Context) {
		clearTmpFile()
		randGif(false)
	})
	randGif(true)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
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
	cmd := exec.Command("ffmpeg", "-i", mp4Path, "-vf", "fps=10", "-s", "2560x1440", "-threads", "1", ffOut)
	_, _ = cmd.CombinedOutput()
}

func clearTmpFile() {
	tmpPath := "/tmp/live-wallpaper"
	if gfile.Exists(tmpPath) {
		ds, _ := gfile.ScanDir(tmpPath, "*")
		for _, v := range ds {
			if gfile.IsDir(v) {
				info, _ := gfile.Stat(v)
				if info.ModTime().Add(time.Hour * 6).Before(time.Now()) {
					_ = gfile.RemoveAll(v)
				}
			}
		}
	}
}

func sendMessage(addr string) {
	conn, err := net.Dial("unix", addr)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = conn.Close()
	}()

	_, err = conn.Write([]byte("trigger"))
	if err != nil {
		panic(err)
	}
}

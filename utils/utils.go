package utils

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func CheckIfFFMPEGExists() bool {
	cmd := exec.Command("ffmpeg", "-version")
	err := cmd.Run()
	if err != nil {
		log.Println(err.Error())
		return false
	}

	return true
}

func GetLogFileWriter() io.Writer {
	today := time.Now().Format("2006-01-02")
	os.Mkdir("logs", 0755)

	logFile, err := os.OpenFile("logs/"+today+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return logFile

}

func DirSize(path string) (int64, error) {
	var size int64
	var mu sync.Mutex

	// Function to calculate size for a given path
	var calculateSize func(string) error
	calculateSize = func(p string) error {
		fileInfo, err := os.Lstat(p)
		if err != nil {
			return err
		}

		// Skip symbolic links to avoid counting them multiple times
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		if fileInfo.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := calculateSize(filepath.Join(p, entry.Name())); err != nil {
					return err
				}
			}
		} else {
			mu.Lock()
			size += fileInfo.Size()
			mu.Unlock()
		}
		return nil
	}

	// Start calculation from the root path
	if err := calculateSize(path); err != nil {
		return 0, err
	}

	return size, nil
}

func GenerateVideoThumbnail(videoPath, outputPath string) error {
	outputDir := filepath.Dir(outputPath)
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Printf("Error creating thumbnail directory %s. Error: %s", outputDir, err.Error())
		return err
	}

	if _, err := os.Stat(outputPath); err == nil {
		return nil
	}

	apw, aph, err := GetVideoAspectRatio(videoPath)
	if err != nil {
		log.Printf("Error getting video aspect ratio from %s. Error: %s", videoPath, err.Error())
		apw = 16
		aph = 9
	}

	maxWidth := 320
	height := maxWidth * aph / apw

	duration, err := GetVideoDuration(videoPath)
	if err != nil {
		log.Printf("Error getting video duration from %s. Error: %s", videoPath, err.Error())
		duration = 0
	}

	cmd := exec.Command("ffmpeg", "-i", videoPath, "-ss",
		fmt.Sprintf("00:00:%02d", min(duration, 8)),
		"-vf", fmt.Sprintf("scale=%d:%d", maxWidth, height), "-frames:v", "1", outputPath)

	return cmd.Run()
}

func GetVideoAspectRatio(videoPath string) (int, int, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=display_aspect_ratio", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	aspectRatio := strings.Split(string(output), ":")
	if len(aspectRatio) != 2 {
		cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)
		output, err := cmd.Output()
		if err != nil {
			return 0, 0, err
		}

		dimensions := strings.Split(strings.ReplaceAll(string(output), "\r", ""), "\n")
		if len(dimensions) != 3 {
			return 0, 0, fmt.Errorf("failed to get video dimensions")
		}

		width, err := strconv.Atoi(dimensions[0])
		if err != nil {
			return 0, 0, err
		}

		height, err := strconv.Atoi(dimensions[1])
		if err != nil {
			return 0, 0, err
		}

		return width, height, nil
	}

	apw, err := strconv.Atoi(aspectRatio[0])
	if err != nil {
		return 0, 0, err
	}

	aph, err := strconv.Atoi(strings.Replace(aspectRatio[1], "\r\n", "", 1))
	if err != nil {
		return 0, 0, err
	}

	return apw, aph, nil
}

func GetVideoDuration(videoPath string) (int, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	seconds := strings.Split(strings.ReplaceAll(string(output), "\r\n", ""), ".")
	duration, err := strconv.Atoi(seconds[0])
	if err != nil {
		return 0, err
	}

	return duration, nil
}

func HashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func GetLocalIp() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if isPrivateIP(ip) {
				return ip, nil
			}
		}
	}

	return nil, errors.New("no IP")
}

func isPrivateIP(ip net.IP) bool {
	var privateIPBlocks []*net.IPNet
	for _, cidr := range []string{
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

package files

import (
	"fmt"

	"github.com/mateo-14/go-http-file-server/utils"
)

func generateThumbnailUrl(thumbnailPath, port string) string {
	localIp, _ := utils.GetLocalIp()

	return fmt.Sprintf("http://%s:%s/thumbnails/%s", localIp, port, thumbnailPath)
}

func generateFileUrl(filePath, port string) string {
	localIp, _ := utils.GetLocalIp()
	return fmt.Sprintf("http://%s:%s/files/%s", localIp, port, filePath)
}

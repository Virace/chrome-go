package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
)

// DownloadProgress 下载进度回调
type DownloadProgress func(downloaded, total int64)

// MultiSourceDownload 多源多线程下载文件
// 将下载任务分配到多个 URL 源，每个源负责不同的分块
func MultiSourceDownload(urls []string, destPath string, threads int, progress DownloadProgress) error {
	if len(urls) == 0 {
		return fmt.Errorf("没有可用的下载链接")
	}

	// 只有一个 URL，使用普通多线程下载
	if len(urls) == 1 {
		return MultiThreadDownload(urls[0], destPath, threads, progress)
	}

	// 获取文件大小（使用第一个 URL）
	totalSize, supportsRange, err := getFileInfo(urls[0])
	if err != nil {
		// 第一个失败，尝试其他
		for i := 1; i < len(urls); i++ {
			totalSize, supportsRange, err = getFileInfo(urls[i])
			if err == nil {
				break
			}
		}
		if err != nil {
			return fmt.Errorf("无法获取文件信息: %w", err)
		}
	}

	// 不支持 Range，降级为单线程
	if !supportsRange || totalSize <= 0 {
		return singleThreadDownload(urls[0], destPath, progress)
	}

	// 创建目标文件
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 预分配文件大小
	if err := file.Truncate(totalSize); err != nil {
		return err
	}

	// 计算每个线程的下载范围
	chunkSize := totalSize / int64(threads)
	var wg sync.WaitGroup
	var downloadedBytes int64
	errChan := make(chan error, threads)

	// 分配任务到多个 URL
	urlCount := len(urls)

	for i := 0; i < threads; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == threads-1 {
			end = totalSize - 1
		}

		// 轮询分配 URL
		url := urls[i%urlCount]

		wg.Add(1)
		go func(url string, start, end int64) {
			defer wg.Done()
			if err := downloadChunk(url, file, start, end, &downloadedBytes, progress, totalSize); err != nil {
				errChan <- err
			}
		}(url, start, end)
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// getFileInfo 获取文件大小和是否支持 Range
func getFileInfo(url string) (int64, bool, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	supportsRange := resp.Header.Get("Accept-Ranges") == "bytes"
	return resp.ContentLength, supportsRange, nil
}

// MultiThreadDownload 多线程下载文件（单源）
func MultiThreadDownload(url, destPath string, threads int, progress DownloadProgress) error {
	totalSize, supportsRange, err := getFileInfo(url)
	if err != nil {
		return err
	}

	if totalSize <= 0 || !supportsRange {
		return singleThreadDownload(url, destPath, progress)
	}

	// 创建目标文件
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 预分配文件大小
	if err := file.Truncate(totalSize); err != nil {
		return err
	}

	// 计算每个线程的下载范围
	chunkSize := totalSize / int64(threads)
	var wg sync.WaitGroup
	var downloadedBytes int64
	errChan := make(chan error, threads)

	for i := 0; i < threads; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == threads-1 {
			end = totalSize - 1
		}

		wg.Add(1)
		go func(start, end int64) {
			defer wg.Done()
			if err := downloadChunk(url, file, start, end, &downloadedBytes, progress, totalSize); err != nil {
				errChan <- err
			}
		}(start, end)
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// downloadChunk 下载文件的一部分
func downloadChunk(url string, file *os.File, start, end int64, downloaded *int64, progress DownloadProgress, total int64) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf := make([]byte, 32*1024) // 32KB buffer
	offset := start

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.WriteAt(buf[:n], offset)
			if writeErr != nil {
				return writeErr
			}
			offset += int64(n)
			newDownloaded := atomic.AddInt64(downloaded, int64(n))
			if progress != nil {
				progress(newDownloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// singleThreadDownload 单线程下载（降级方案）
func singleThreadDownload(url, destPath string, progress DownloadProgress) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// FormatBytes 格式化字节数
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

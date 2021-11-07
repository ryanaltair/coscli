package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io/ioutil"
	"os"
	"regexp"
)

func MatchBucketPattern(buckets []cos.Bucket, pattern string, include bool) []cos.Bucket {
	res := make([]cos.Bucket, 0)
	for _, b := range buckets {
		match, _ := regexp.Match(pattern, []byte(b.Name))
		if !include {
			match = !match
		}
		if match {
			res = append(res, b)
		}
	}
	return res
}

func MatchCosPattern(objects []cos.Object, pattern string, include bool) []cos.Object {
	res := make([]cos.Object, 0)
	for _, o := range objects {
		match, _ := regexp.Match(pattern, []byte(o.Key))
		if !include {
			match = !match
		}
		if match {
			res = append(res, o)
		}
	}
	return res
}

func MatchUploadPattern(uploads []UploadInfo, pattern string, include bool) []UploadInfo {
	res := make([]UploadInfo, 0)
	for _, u := range uploads {
		match, _ := regexp.Match(pattern, []byte(u.Key))
		if !include {
			match = !match
		}
		if match {
			res = append(res, u)
		}
	}
	return res
}

func MatchPattern(strs []string, pattern string, include bool) []string {
	res := make([]string, 0)
	re := regexp.MustCompile(pattern)
	for _, s := range strs {
		match := re.MatchString(s)
		if !include {
			match = !match
		}
		if match {
			res = append(res, s)
		}
	}
	return res
}

func GetBucketsList(c *cos.Client, limit int, include string, exclude string) (buckets []cos.Bucket) {
	res, _, err := c.Service.Get(context.Background())
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	buckets = res.Buckets
	if len(include) > 0 {
		buckets = MatchBucketPattern(buckets, include, true)
	}
	if len(exclude) > 0 {
		buckets = MatchBucketPattern(buckets, exclude, false)
	}

	if limit > 0 {
		var l int
		if limit > len(buckets) {
			l = len(buckets)
		} else {
			l = limit
		}
		return buckets[:l]
	} else {
		return buckets
	}
}

func GetObjectsList(c *cos.Client, prefix string, limit int, include string, exclude string) (dirs []string, objects []cos.Object) {
	opt := &cos.BucketGetOptions{
		Prefix:       prefix,
		Delimiter:    "/",
		EncodingType: "",
		Marker:       "",
		MaxKeys:      limit,
	}

	isTruncated := true
	marker := ""
	for isTruncated {
		opt.Marker = marker

		res, _, err := c.Bucket.Get(context.Background(), opt)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		dirs = append(dirs, res.CommonPrefixes...)
		objects = append(objects, res.Contents...)

		if limit > 0 {
			isTruncated = false
		} else {
			isTruncated = res.IsTruncated
			marker = res.NextMarker
		}
	}

	if len(include) > 0 {
		objects = MatchCosPattern(objects, include, true)
		dirs = MatchPattern(dirs, include, true)
	}
	if len(exclude) > 0 {
		objects = MatchCosPattern(objects, exclude, false)
		dirs = MatchPattern(dirs, exclude, false)
	}

	return dirs, objects
}

func GetObjectsListRecursive(c *cos.Client, prefix string, limit int, include string, exclude string) (objects []cos.Object) {
	opt := &cos.BucketGetOptions{
		Prefix:       prefix,
		Delimiter:    "",
		EncodingType: "",
		Marker:       "",
		MaxKeys:      limit,
	}

	isTruncated := true
	marker := ""
	for isTruncated {
		opt.Marker = marker

		res, _, err := c.Bucket.Get(context.Background(), opt)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		objects = append(objects, res.Contents...)

		if limit > 0 {
			isTruncated = false
		} else {
			isTruncated = res.IsTruncated
			marker = res.NextMarker
		}
	}

	if len(include) > 0 {
		objects = MatchCosPattern(objects, include, true)
	}
	if len(exclude) > 0 {
		objects = MatchCosPattern(objects, exclude, false)
	}

	return objects
}

func GetLocalFilesList(localPath string, include string, exclude string) (dirs []string, files []string) {
	fileInfos, err := ioutil.ReadDir(localPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, f := range fileInfos {
		fileName := localPath + "/" + f.Name()
		fileName = fileName[len(localPath)+1:]
		if f.IsDir() {
			dirs = append(dirs, fileName)
		} else {
			files = append(files, fileName)
		}
	}

	if len(include) > 0 {
		files = MatchPattern(files, include, true)
		dirs = MatchPattern(dirs, include, true)
	}
	if len(exclude) > 0 {
		files = MatchPattern(files, exclude, true)
		dirs = MatchPattern(dirs, exclude, false)
	}

	return dirs, files
}

func GetLocalFilesListRecursive(localPath string, include string, exclude string) (files []string) {
	// bfs遍历文件夹
	var dirs []string
	dirs = append(dirs, localPath)
	for len(dirs) > 0 {
		dirName := dirs[0]
		dirs = dirs[1:]

		fileInfos, err := ioutil.ReadDir(dirName)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		for _, f := range fileInfos {
			fileName := dirName + "/" + f.Name()
			if f.IsDir() {
				dirs = append(dirs, fileName)
			} else {
				fileName = fileName[len(localPath)+1:]
				files = append(files, fileName)
			}
		}
	}

	if len(include) > 0 {
		files = MatchPattern(files, include, true)
	}
	if len(exclude) > 0 {
		files = MatchPattern(files, exclude, false)
	}

	return files
}

// 疑似无法返回正确结果
// res.CommonPrefix无法正确获得
func GetUploadsList(c *cos.Client, prefix string, limit int, include string, exclude string) (dirs []string, uploads []UploadInfo) {
	opt := &cos.ListMultipartUploadsOptions{
		Delimiter:      "/",
		EncodingType:   "",
		Prefix:         prefix,
		MaxUploads:     limit,
		KeyMarker:      "",
		UploadIDMarker: "",
	}

	isTruncated := true
	keyMarker := ""
	uploadIDMarker := ""
	for isTruncated {
		opt.KeyMarker = keyMarker
		opt.UploadIDMarker = uploadIDMarker

		res, _, err := c.Bucket.ListMultipartUploads(context.Background(), opt)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		dirs = append(dirs, res.CommonPrefixes...)
		for _, u := range res.Uploads {
			uploads = append(uploads, UploadInfo{
				Key:       u.Key,
				UploadID:  u.UploadID,
				Initiated: u.Initiated,
			})
		}

		if limit > 0 {
			isTruncated = false
		} else {
			isTruncated = res.IsTruncated
			keyMarker = res.NextKeyMarker
			uploadIDMarker = res.NextUploadIDMarker
		}
	}

	if len(include) > 0 {
		uploads = MatchUploadPattern(uploads, include, true)
	}
	if len(exclude) > 0 {
		uploads = MatchUploadPattern(uploads, exclude, false)
	}

	return dirs, uploads
}

func GetUploadsListRecursive(c *cos.Client, prefix string, limit int, include string, exclude string) (uploads []UploadInfo) {
	opt := &cos.ListMultipartUploadsOptions{
		Delimiter:      "",
		EncodingType:   "",
		Prefix:         prefix,
		MaxUploads:     limit,
		KeyMarker:      "",
		UploadIDMarker: "",
	}

	isTruncated := true
	keyMarker := ""
	uploadIDMarker := ""
	for isTruncated {
		opt.KeyMarker = keyMarker
		opt.UploadIDMarker = uploadIDMarker

		res, _, err := c.Bucket.ListMultipartUploads(context.Background(), opt)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		for _, u := range res.Uploads {
			uploads = append(uploads, UploadInfo{
				Key:       u.Key,
				UploadID:  u.UploadID,
				Initiated: u.Initiated,
			})
		}

		if limit > 0 {
			isTruncated = false
		} else {
			isTruncated = res.IsTruncated
			keyMarker = res.NextKeyMarker
			uploadIDMarker = res.NextUploadIDMarker
		}
	}

	if len(include) > 0 {
		uploads = MatchUploadPattern(uploads, include, true)
	}
	if len(exclude) > 0 {
		uploads = MatchUploadPattern(uploads, exclude, false)
	}

	return uploads
}
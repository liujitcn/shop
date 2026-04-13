package biz

import (
	"errors"
	"slices"

	"shop/api/gen/go/base"

	"github.com/go-kratos/kratos/v2/log"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/kratos-kit/oss"
	"github.com/liujitcn/kratos-kit/sdk"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type FileCase struct {
	oss.OSS
}

// NewFileCase 创建文件业务实例。
func NewFileCase(
	oss oss.OSS,
) *FileCase {
	sdk.Runtime.SetOSS(oss)
	return &FileCase{
		OSS: oss,
	}
}

// MultiUploadFile 批量上传文件。
func (c *FileCase) MultiUploadFile(req *base.MultiUploadFileRequest) (*base.MultiUploadFileResponse, error) {
	files := make([]*base.FileInfo, 0)
	uploadFiles := req.GetFiles()
	// 未传入上传文件时，直接返回错误。
	if len(uploadFiles) == 0 {
		return nil, errors.New("no upload file")
	}
	for _, item := range uploadFiles {
		url, err := c.UploadByByte(item.GetName(), item.GetPath(), item.GetContent())
		if err != nil {
			log.Error("MultiUploadFile err:", err.Error())
			return nil, errors.New("文件上传失败")
		}
		files = append(files, &base.FileInfo{
			Url:     url,
			Name:    item.GetName(),
			Extname: item.GetExtname(),
		})
	}
	return &base.MultiUploadFileResponse{Files: files}, nil
}

// UploadFile 上传单个文件。
func (c *FileCase) UploadFile(req *base.UploadFileInfo) (*base.FileInfo, error) {
	url, err := c.UploadByByte(req.GetName(), req.GetPath(), req.GetContent())
	if err != nil {
		log.Error("UploadFile err:", err.Error())
		return nil, errors.New("文件上传失败")
	}
	return &base.FileInfo{
		Url:     url,
		Name:    req.GetName(),
		Extname: req.GetExtname(),
	}, nil
}

// DownloadFile 下载文件内容。
func (c *FileCase) DownloadFile(req *base.DownloadFileRequest) (*wrapperspb.BytesValue, error) {
	fileByte, err := c.GetFileByte(req.GetPath())
	if err != nil {
		log.Error("DownloadFile err:", err.Error())
		return nil, errors.New("文件下载失败")
	}
	return &wrapperspb.BytesValue{Value: fileByte}, nil
}

// MultiDeleteFileByString 按字符串数组配置删除历史文件。
func (c *FileCase) MultiDeleteFileByString(oldFile string, newFile []string) {
	c.MultiDeleteFile(_string.ConvertJsonStringToStringArray(oldFile), newFile)
}

// MultiDeleteFile 批量删除不再使用的旧文件。
func (c *FileCase) MultiDeleteFile(oldFile, newFile []string) {
	for _, item := range oldFile {
		// 新文件列表为空或未包含旧文件时，删除当前旧文件。
		if len(newFile) == 0 || !slices.Contains(newFile, item) {
			err := c.OSS.DeleteFile(item)
			// 单个旧文件删除失败时，只记录日志继续处理剩余文件。
			if err != nil {
				log.Error("MultiDeleteFile err:", err.Error())
			}
		}
	}
}

// DeleteFile 删除单个旧文件。
func (c *FileCase) DeleteFile(oldFile string, newFile string) {
	// 新旧文件不一致时，删除历史文件资源。
	if newFile == "" || oldFile != newFile {
		// 删除旧文件
		err := c.OSS.DeleteFile(oldFile)
		// 删除单个旧文件失败时，只记录日志不阻断调用方流程。
		if err != nil {
			log.Error("DeleteFile err:", err.Error())
		}
	}
}

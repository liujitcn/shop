package base

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path"
	_const "shop/pkg/const"
	"strconv"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/kratos/v2/transport/http/binding"
	"github.com/liujitcn/go-utils/id"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the Nms package it is being compiled against.
var _ = new(context.Context)
var _ = binding.EncodeURL

const _ = http.SupportPackageIsVersion1

const OperationFileServiceDownloadFile = "/base.v1.FileService/DownloadFile"
const OperationFileServiceMultiUploadFile = "/base.v1.FileService/MultiUploadFile"
const OperationFileServiceUploadFile = "/base.v1.FileService/UploadFile"

type FileServiceHTTPServer interface {
	// DownloadFile 下载文件
	DownloadFile(context.Context, *basev1.DownloadFileRequest) (*wrapperspb.BytesValue, error)
	// MultiUploadFile 多个文件上传
	MultiUploadFile(context.Context, *basev1.MultiUploadFileRequest) (*basev1.MultiUploadFileResponse, error)
	// UploadFile 单个文件上传
	UploadFile(context.Context, *basev1.UploadFileRequest) (*basev1.FileInfo, error)
}

func RegisterFileServiceHTTPServer(s *http.Server, srv FileServiceHTTPServer) {
	r := s.Route("/")
	r.POST("/api/v1/base/file/multi", _FileService_MultiUploadFile0_HTTP_Handler(srv))
	r.POST("/api/v1/base/file", _FileService_UploadFile0_HTTP_Handler(srv))
	r.GET("/api/v1/base/file", _FileService_DownloadFile0_HTTP_Handler(srv))
}

func _FileService_MultiUploadFile0_HTTP_Handler(srv FileServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in basev1.MultiUploadFileRequest
		r := ctx.Request()
		if r.MultipartForm == nil {
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				return err
			}
		}
		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			for _, item := range r.MultipartForm.File {
				fhs := item[0]
				formFile, err := fhs.Open()
				if err != nil {
					return err
				}
				contentType := fhs.Header.Get("Content-Type")
				var uploadFileInfo *basev1.UploadFileInfo
				uploadFileInfo, err = convertUploadFileInfo(formFile, r.FormValue("fileType"), contentType, fhs.Filename)
				if err != nil {
					return err
				}
				in.Files = append(in.Files, uploadFileInfo)
			}
		}
		http.SetOperation(ctx, OperationFileServiceMultiUploadFile)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.MultiUploadFile(ctx, req.(*basev1.MultiUploadFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*basev1.MultiUploadFileResponse)
		return ctx.Result(200, reply)
	}
}

func _FileService_UploadFile0_HTTP_Handler(srv FileServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		r := ctx.Request()
		// 修改获取文件内容方法
		formFile, header, err := r.FormFile("file")
		if err != nil {
			return err
		}
		contentType := header.Header.Get("Content-Type")
		var uploadFileInfo *basev1.UploadFileInfo
		uploadFileInfo, err = convertUploadFileInfo(formFile, r.FormValue("fileType"), contentType, header.Filename)
		if err != nil {
			return err
		}
		in := &basev1.UploadFileRequest{File: uploadFileInfo}
		http.SetOperation(ctx, OperationFileServiceUploadFile)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.UploadFile(ctx, req.(*basev1.UploadFileRequest))
		})
		var out interface{}
		out, err = h(ctx, in)
		if err != nil {
			return err
		}
		reply := out.(*basev1.FileInfo)
		return ctx.Result(200, reply)
	}
}

func _FileService_DownloadFile0_HTTP_Handler(srv FileServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in basev1.DownloadFileRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationFileServiceDownloadFile)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DownloadFile(ctx, req.(*basev1.DownloadFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*wrapperspb.BytesValue)
		filename := in.GetName()
		if len(filename) == 0 {
			filename = path.Base(in.GetPath())
		}
		// 设置响应头，支持文件下载
		ctx.Response().Header().Set("Content-Type", "application/octet-stream")
		ctx.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		ctx.Response().Header().Set("Content-Length", strconv.Itoa(len(reply.Value)))

		// 直接写入二进制数据
		_, err = ctx.Response().Write(reply.Value)
		if err != nil {
			return err
		}

		return nil
	}
}

func convertUploadFileInfo(multipartFile multipart.File, fileType, contentType, fileName string) (*basev1.UploadFileInfo, error) {
	defer func(multipartFile multipart.File) {
		err := multipartFile.Close()
		if err != nil {
			log.Errorf("form file close err: %v", err)
		}
	}(multipartFile)

	b := new(strings.Builder)
	_, err := io.Copy(b, multipartFile)
	if err != nil {
		return nil, err
	}
	filePath := fmt.Sprintf("/%s", _const.BASE_PATH)
	if len(fileType) != 0 {
		filePath += "/" + fileType
	}
	var extname string
	contentTypes := strings.Split(contentType, "/")
	if len(contentTypes) != 2 {
		filePath += "/files"
		filenames := strings.Split(fileName, ".")
		if len(filenames) > 1 {
			extname = filenames[1]
		}
	} else {
		extname = contentTypes[1]
		switch contentTypes[0] {
		case "image":
			filePath += "/images"
			break
		case "video":
			filePath += "/videos"
			break
		case "audio":
			filePath += "/audios"
			break
		case "application", "text":
			filePath += "/docs"
			break
		default:
			filePath += "/files"
			break
		}
	}

	datePath := time.Now().Format("2006/01/02")
	return &basev1.UploadFileInfo{
		Name:    fmt.Sprintf("%d.%s", id.GenSnowflakeID(), extname),
		Extname: extname,
		Path:    fmt.Sprintf("%s/%s", filePath, datePath),
		Content: []byte(b.String()),
	}, nil
}

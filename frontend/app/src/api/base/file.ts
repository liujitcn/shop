import { http } from '@/utils/http'
import type { DownloadFileRequest, MultiUploadFileResponse, FileInfo } from "@/rpc/base/file";

const FILE_URL = '/file'
const STATIC_API_BASE_PATH = import.meta.env.VITE_APP_STATIC_API

/** 文件服务 */
export class FileServiceImpl {
  /** 多个文件上传 */
  MultiUploadFile(files: File[], fileType: string): Promise<MultiUploadFileResponse> {
    const formData = new FormData()
    files.map((file) => {
      formData.append(file.name, file)
    })
    // 透传静态资源基础目录，保持和管理端上传行为一致
    formData.append('basePath', STATIC_API_BASE_PATH)
    formData.append('fileType', fileType)
    return http<MultiUploadFileResponse>({
      url: `${FILE_URL}/multi`,
      method: 'POST',
      data: formData,
      header: {
        'Content-Type': 'multipart/form-data',
      },
    })
  }
  /** 单个文件上传 */
  UploadFile(file: File, fileType: string): Promise<FileInfo> {
    const formData = new FormData()
    formData.append('file', file)
    // 透传静态资源基础目录，保持和管理端上传行为一致
    formData.append('basePath', STATIC_API_BASE_PATH)
    formData.append('fileType', fileType)
    return http<FileInfo>({
      url: `${FILE_URL}`,
      method: 'POST',
      data: formData,
      header: {
        'Content-Type': 'multipart/form-data',
      },
    })
  }
  /** 下载文件 */

  async DownloadFile(file: string, fileName: string) {
    try {
      const response = await http<any>({
        url: `${FILE_URL}`,
        method: 'GET',
        data: {
          name: fileName,
          path: file,
        } as DownloadFileRequest,
        responseType: 'blob', // 明确要求返回 Blob 类型
      })
      // 获取文件名
      const contentDisposition = response.headers['content-disposition']
      let downloadFilename = fileName
      if (contentDisposition) {
        const match = contentDisposition.match(/filename="?(.+?)"?$/)
        if (match && match[1]) {
          downloadFilename = match[1]
        }
      }

      // 创建下载链接
      const url = window.URL.createObjectURL(new Blob([response.data]))
      const a = document.createElement('a')
      a.href = url
      a.download = downloadFilename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
    } catch (error) {
      console.error('下载错误:', error)
    }
  }
}

export const defFileService = new FileServiceImpl()

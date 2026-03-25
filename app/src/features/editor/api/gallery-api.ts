import apiClient from '@/lib/api-client'

export interface GalleryAsset {
  id: string
  key: string
  filename: string
  contentType: string
  size: number
  createdAt: string
}

export interface GalleryListResponse {
  items: GalleryAsset[]
  total: number
  page: number
  perPage: number
}

export const galleryApi = {
  async list(page = 1, perPage = 20): Promise<GalleryListResponse> {
    const response = await apiClient.get<GalleryListResponse>('/workspace/gallery', {
      params: { page, perPage },
    })
    return response.data
  },

  async search(q: string, page = 1, perPage = 20): Promise<GalleryListResponse> {
    const response = await apiClient.get<GalleryListResponse>('/workspace/gallery/search', {
      params: { q, page, perPage },
    })
    return response.data
  },

  async upload(file: File): Promise<GalleryAsset> {
    const form = new FormData()
    form.append('file', file)
    const response = await apiClient.post<GalleryAsset>('/workspace/gallery', form, {
      headers: { 'Content-Type': undefined },
    })
    return response.data
  },

  async delete(key: string): Promise<void> {
    await apiClient.delete('/workspace/gallery', { params: { key } })
  },

  async getURL(key: string): Promise<string> {
    const response = await apiClient.get<{ url: string }>('/workspace/gallery/url', {
      params: { key },
    })
    return response.data.url
  },

  /**
   * Resolves a gallery key to a URL safe for use in <img src>.
   * For local storage the backend returns a /serve endpoint URL which requires
   * auth headers — browsers cannot send those from img tags. In that case we
   * fetch the image via axios (auth headers included) and return a blob URL.
   * For S3 the backend returns a presigned URL that is directly loadable.
   */
  async getSrc(key: string): Promise<string> {
    const url = await galleryApi.getURL(key)
    if (url.includes('/workspace/gallery/serve')) {
      const response = await apiClient.get<Blob>('/workspace/gallery/serve', {
        params: { key },
        responseType: 'blob',
      })
      return URL.createObjectURL(response.data)
    }
    return url
  },
}

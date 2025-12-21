import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import axios from 'axios'
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'

// Mock axios
vi.mock('axios', () => {
  const mockInterceptors = {
    request: {
      use: vi.fn(),
      handlers: [] as Array<{ fulfilled: Function; rejected: Function }>
    },
    response: {
      use: vi.fn(),
      handlers: [] as Array<{ fulfilled: Function; rejected: Function }>
    }
  }

  // Capture interceptor handlers when use is called
  mockInterceptors.request.use.mockImplementation((fulfilled, rejected) => {
    mockInterceptors.request.handlers.push({ fulfilled, rejected })
    return mockInterceptors.request.handlers.length - 1
  })

  mockInterceptors.response.use.mockImplementation((fulfilled, rejected) => {
    mockInterceptors.response.handlers.push({ fulfilled, rejected })
    return mockInterceptors.response.handlers.length - 1
  })

  const mockAxiosInstance = {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    interceptors: mockInterceptors
  }

  return {
    default: {
      create: vi.fn(() => mockAxiosInstance)
    }
  }
})

describe('api service', () => {
  let mockCreate: ReturnType<typeof vi.fn>
  let mockAxiosInstance: any
  let requestInterceptor: { fulfilled: Function; rejected: Function }
  let responseInterceptor: { fulfilled: Function; rejected: Function }

  beforeEach(async () => {
    vi.clearAllMocks()

    // Reset window location mock
    delete (window as any).location
    ;(window as any).location = {
      pathname: '/dashboard',
      href: ''
    }

    // Reset sessionStorage mock
    const sessionStorageMock = {
      store: {} as Record<string, string>,
      getItem: vi.fn((key: string) => sessionStorageMock.store[key] || null),
      setItem: vi.fn((key: string, value: string) => {
        sessionStorageMock.store[key] = value
      }),
      removeItem: vi.fn((key: string) => {
        delete sessionStorageMock.store[key]
      }),
      clear: vi.fn(() => {
        sessionStorageMock.store = {}
      })
    }
    Object.defineProperty(window, 'sessionStorage', {
      value: sessionStorageMock,
      writable: true
    })

    // Reset axios mock
    vi.resetModules()

    // Setup mock to capture interceptors
    const mockInterceptors = {
      request: {
        use: vi.fn(),
        handlers: [] as Array<{ fulfilled: Function; rejected: Function }>
      },
      response: {
        use: vi.fn(),
        handlers: [] as Array<{ fulfilled: Function; rejected: Function }>
      }
    }

    mockInterceptors.request.use.mockImplementation((fulfilled: Function, rejected: Function) => {
      mockInterceptors.request.handlers.push({ fulfilled, rejected })
      return mockInterceptors.request.handlers.length - 1
    })

    mockInterceptors.response.use.mockImplementation((fulfilled: Function, rejected: Function) => {
      mockInterceptors.response.handlers.push({ fulfilled, rejected })
      return mockInterceptors.response.handlers.length - 1
    })

    mockAxiosInstance = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn(),
      interceptors: mockInterceptors
    }

    mockCreate = vi.fn(() => mockAxiosInstance)
    vi.mocked(axios.create).mockImplementation(mockCreate)

    // Import the api module to trigger interceptor setup
    await import('./api')

    // Get the interceptor handlers
    requestInterceptor = mockInterceptors.request.handlers[0]
    responseInterceptor = mockInterceptors.response.handlers[0]
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('axios instance creation', () => {
    it('creates axios instance with correct baseURL', () => {
      expect(mockCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          baseURL: '/api/v1'
        })
      )
    })

    it('creates axios instance with timeout', () => {
      expect(mockCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          timeout: 10000
        })
      )
    })

    it('creates axios instance with JSON content type', () => {
      expect(mockCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          headers: {
            'Content-Type': 'application/json'
          }
        })
      )
    })

    it('creates axios instance with credentials enabled', () => {
      expect(mockCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          withCredentials: true
        })
      )
    })
  })

  describe('request interceptor', () => {
    it('passes config through unchanged', () => {
      const config = { url: '/test', method: 'get' } as InternalAxiosRequestConfig
      const result = requestInterceptor.fulfilled(config)
      expect(result).toBe(config)
    })

    it('rejects errors', async () => {
      const error = new Error('Request error')
      await expect(requestInterceptor.rejected(error)).rejects.toThrow('Request error')
    })
  })

  describe('response interceptor', () => {
    it('extracts data from response', () => {
      const response = { data: { name: 'test' }, status: 200 } as AxiosResponse
      const result = responseInterceptor.fulfilled(response)
      expect(result).toEqual({ name: 'test' })
    })

    it('handles 401 error by redirecting to login', async () => {
      const error = {
        response: { status: 401, data: { error: 'Unauthorized' } },
        message: 'Unauthorized'
      }

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      await expect(responseInterceptor.rejected(error)).rejects.toBe(error)

      expect(window.sessionStorage.setItem).toHaveBeenCalledWith('redirectAfterLogin', '/dashboard')
      expect(window.location.href).toBe('/login')

      consoleSpy.mockRestore()
    })

    it('does not redirect if already on login page', async () => {
      ;(window as any).location.pathname = '/login'

      const error = {
        response: { status: 401, data: { error: 'Unauthorized' } },
        message: 'Unauthorized'
      }

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      await expect(responseInterceptor.rejected(error)).rejects.toBe(error)

      expect(window.sessionStorage.setItem).not.toHaveBeenCalled()
      expect(window.location.href).toBe('')

      consoleSpy.mockRestore()
    })

    it('logs API error with response data', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const error = {
        response: { status: 500, data: { error: 'Server error' } },
        message: 'Internal Server Error'
      }

      await expect(responseInterceptor.rejected(error)).rejects.toBe(error)

      expect(consoleSpy).toHaveBeenCalledWith('API Error:', { error: 'Server error' })

      consoleSpy.mockRestore()
    })

    it('logs API error with message when no response data', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const error = {
        message: 'Network Error'
      }

      await expect(responseInterceptor.rejected(error)).rejects.toBe(error)

      expect(consoleSpy).toHaveBeenCalledWith('API Error:', 'Network Error')

      consoleSpy.mockRestore()
    })

    it('handles non-401 errors without redirect', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const error = {
        response: { status: 404, data: { error: 'Not found' } },
        message: 'Not Found'
      }

      await expect(responseInterceptor.rejected(error)).rejects.toBe(error)

      expect(window.sessionStorage.setItem).not.toHaveBeenCalled()
      expect(window.location.href).toBe('')

      consoleSpy.mockRestore()
    })

    it('handles 403 errors without redirect', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const error = {
        response: { status: 403, data: { error: 'Forbidden' } },
        message: 'Forbidden'
      }

      await expect(responseInterceptor.rejected(error)).rejects.toBe(error)

      expect(window.sessionStorage.setItem).not.toHaveBeenCalled()
      expect(window.location.href).toBe('')

      consoleSpy.mockRestore()
    })
  })
})

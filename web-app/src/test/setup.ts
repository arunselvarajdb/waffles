import { vi, beforeAll, afterAll } from 'vitest'
import { config } from '@vue/test-utils'

// Suppress console.error during tests to reduce noise from expected error scenarios
// The actual error handling is still tested - we just don't print the messages
const originalConsoleError = console.error
beforeAll(() => {
  console.error = vi.fn()
})
afterAll(() => {
  console.error = originalConsoleError
})

// Mock window.location
Object.defineProperty(window, 'location', {
  value: {
    href: 'http://localhost:3000',
    pathname: '/',
    assign: vi.fn(),
    replace: vi.fn(),
  },
  writable: true,
})

// Mock sessionStorage
const sessionStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
Object.defineProperty(window, 'sessionStorage', { value: sessionStorageMock })

// Mock fetch globally
global.fetch = vi.fn()

// Suppress Vue warnings in tests
config.global.config.warnHandler = () => {}

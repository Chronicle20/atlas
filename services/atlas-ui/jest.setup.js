import '@testing-library/jest-dom'

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useRouter() {
    return {
      push: jest.fn(),
      replace: jest.fn(),
      prefetch: jest.fn(),
      back: jest.fn(),
      forward: jest.fn(),
      refresh: jest.fn(),
    }
  },
  useSearchParams() {
    return new URLSearchParams()
  },
  usePathname() {
    return ''
  },
}))

// Mock IntersectionObserver
global.IntersectionObserver = class IntersectionObserver {
  constructor() {}
  disconnect() {}
  observe() {}
  unobserve() {}
}

// Mock ResizeObserver
global.ResizeObserver = class ResizeObserver {
  constructor() {}
  disconnect() {}
  observe() {}
  unobserve() {}
}

// Mock HTMLFormElement.requestSubmit for JSDOM
Object.defineProperty(HTMLFormElement.prototype, 'requestSubmit', {
  value: function(submitter) {
    if (submitter) {
      submitter.click()
    } else {
      this.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    }
  },
  writable: true,
  configurable: true,
})

// Mock hasPointerCapture for JSDOM
HTMLElement.prototype.hasPointerCapture = jest.fn(() => false)
HTMLElement.prototype.setPointerCapture = jest.fn()
HTMLElement.prototype.releasePointerCapture = jest.fn()

// Mock window.location.href for error boundary tests
const originalLocation = window.location
delete window.location
window.location = { 
  href: '', 
  reload: jest.fn(),
  assign: jest.fn(),
  replace: jest.fn()
}

// Suppress console.error for expected errors in tests
const originalError = console.error
console.error = (...args) => {
  if (
    typeof args[0] === 'string' &&
    (args[0].includes('Warning: ReactDOM.render is no longer supported') ||
     args[0].includes('Error: Not implemented: navigation') ||
     args[0].includes('The above error occurred'))
  ) {
    return
  }
  // Suppress React error boundary test errors
  if (args[0] && args[0].toString && args[0].toString().includes('Test error')) {
    return
  }
  originalError.call(console, ...args)
}
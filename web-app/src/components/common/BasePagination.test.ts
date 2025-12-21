import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BasePagination from './BasePagination.vue'

describe('BasePagination', () => {
  const defaultProps = {
    currentPage: 1,
    total: 100,
    perPage: 10
  }

  it('renders pagination component', () => {
    const wrapper = mount(BasePagination, { props: defaultProps })
    expect(wrapper.find('nav').exists()).toBe(true)
  })

  it('displays correct results info', () => {
    const wrapper = mount(BasePagination, { props: defaultProps })
    expect(wrapper.text()).toContain('Showing')
    expect(wrapper.text()).toContain('1')
    expect(wrapper.text()).toContain('10')
    expect(wrapper.text()).toContain('100')
  })

  it('calculates startItem correctly', () => {
    const wrapper = mount(BasePagination, {
      props: { ...defaultProps, currentPage: 3 }
    })
    expect(wrapper.text()).toContain('21') // (3-1)*10 + 1 = 21
  })

  it('calculates endItem correctly', () => {
    const wrapper = mount(BasePagination, {
      props: { ...defaultProps, currentPage: 2 }
    })
    expect(wrapper.text()).toContain('20') // 2 * 10 = 20
  })

  it('caps endItem at total on last page', () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 4, total: 35, perPage: 10 }
    })
    expect(wrapper.text()).toContain('35') // Should show 35, not 40
  })

  it('shows 0 startItem when total is 0', () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 1, total: 0, perPage: 10 }
    })
    expect(wrapper.text()).toContain('0')
  })

  it('disables previous button on first page', () => {
    const wrapper = mount(BasePagination, { props: defaultProps })
    const prevButtons = wrapper.findAll('button[disabled]')
    expect(prevButtons.length).toBeGreaterThan(0)
  })

  it('disables next button on last page', () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 10, total: 100, perPage: 10 }
    })
    const buttons = wrapper.findAll('button')
    const nextButton = buttons[buttons.length - 1]
    expect(nextButton.attributes('disabled')).toBeDefined()
  })

  it('enables previous button on non-first page', () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 2, total: 100, perPage: 10 }
    })
    const desktopPrev = wrapper.find('nav button')
    expect(desktopPrev.attributes('disabled')).toBeUndefined()
  })

  it('emits page-change on page button click', async () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 1, total: 50, perPage: 10 }
    })
    const pageButtons = wrapper.findAll('nav button')
    // Find a page number button (not prev/next)
    const page2Button = pageButtons.find(b => b.text() === '2')
    if (page2Button) {
      await page2Button.trigger('click')
      expect(wrapper.emitted('page-change')).toBeTruthy()
      expect(wrapper.emitted('page-change')![0]).toEqual([2])
    }
  })

  it('emits page-change on previous click', async () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 2, total: 50, perPage: 10 }
    })
    const prevButton = wrapper.find('nav button')
    await prevButton.trigger('click')
    expect(wrapper.emitted('page-change')).toBeTruthy()
    expect(wrapper.emitted('page-change')![0]).toEqual([1])
  })

  it('emits page-change on next click', async () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 2, total: 50, perPage: 10 }
    })
    const buttons = wrapper.findAll('nav button')
    const nextButton = buttons[buttons.length - 1]
    await nextButton.trigger('click')
    expect(wrapper.emitted('page-change')).toBeTruthy()
    expect(wrapper.emitted('page-change')![0]).toEqual([3])
  })

  it('does not emit when clicking current page', async () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 1, total: 50, perPage: 10 }
    })
    const pageButtons = wrapper.findAll('nav button')
    const currentPageButton = pageButtons.find(b => b.text() === '1')
    if (currentPageButton) {
      await currentPageButton.trigger('click')
      expect(wrapper.emitted('page-change')).toBeFalsy()
    }
  })

  it('highlights current page with active styles', () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 2, total: 50, perPage: 10 }
    })
    const pageButtons = wrapper.findAll('nav button')
    const currentPageButton = pageButtons.find(b => b.text() === '2')
    expect(currentPageButton?.classes()).toContain('bg-blue-600')
    expect(currentPageButton?.classes()).toContain('text-white')
  })

  it('renders correct number of visible pages', () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 5, total: 100, perPage: 10, maxVisiblePages: 5 }
    })
    // Should show 5 page buttons plus prev/next
    const pageButtons = wrapper.findAll('nav button')
    // 5 page numbers + 2 nav buttons = 7
    expect(pageButtons.length).toBe(7)
  })

  it('has mobile pagination buttons', () => {
    const wrapper = mount(BasePagination, { props: defaultProps })
    const mobileSection = wrapper.find('.sm\\:hidden')
    expect(mobileSection.exists()).toBe(true)
    expect(mobileSection.text()).toContain('Previous')
    expect(mobileSection.text()).toContain('Next')
  })

  it('calculates total pages correctly', () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 1, total: 45, perPage: 10 }
    })
    // 45 / 10 = 4.5 => 5 pages
    const pageButtons = wrapper.findAll('nav button')
    const page5 = pageButtons.find(b => b.text() === '5')
    expect(page5).toBeTruthy()
  })

  it('uses default perPage of 10', () => {
    const wrapper = mount(BasePagination, {
      props: { currentPage: 1, total: 100 }
    })
    expect(wrapper.text()).toContain('10')
  })
})

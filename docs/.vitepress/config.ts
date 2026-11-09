import { defineConfig } from 'vitepress'

const repo = 'https://github.com/dustinmichels/resume-builder'
const dmg = `${repo}/releases/latest/download/Resume-Builder.dmg`

// Served from GitHub Pages at https://dustinmichels.github.io/resume-builder/.
export default defineConfig({
  base: '/resume-builder/',
  title: 'Resume Builder',
  description:
    'A local resume editor: edit on the page, see the exact PDF as you type, and check what applicant tracking systems will read.',
  cleanUrls: true,
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/resume-builder/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#7a1a4a' }],
  ],
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Guide', link: '/guide/install', activeMatch: '/guide/' },
      { text: 'Developers', link: '/developers/', activeMatch: '/developers/' },
      { text: 'Download', link: dmg },
      { text: 'Releases', link: `${repo}/releases` },
    ],
    sidebar: {
      '/guide/': [
        {
          text: 'Get started',
          items: [
            { text: 'Install', link: '/guide/install' },
            { text: 'Tour of the editor', link: '/guide/tour' },
          ],
        },
        {
          text: 'Use it',
          items: [
            { text: 'Editing', link: '/guide/editing' },
            { text: 'Templates & style', link: '/guide/templates' },
            { text: 'Cover letters', link: '/guide/cover-letters' },
            { text: 'Readability checks', link: '/guide/checks' },
            { text: 'Downloads & sync', link: '/guide/downloads' },
            { text: 'Your files & versions', link: '/guide/files' },
          ],
        },
      ],
      '/developers/': [
        {
          text: 'Developers',
          items: [
            { text: 'Run from source', link: '/developers/' },
            { text: 'Architecture', link: '/developers/architecture' },
            { text: 'Content format', link: '/developers/content-format' },
            { text: 'Writing templates', link: '/developers/templates' },
            { text: 'Editing in Typst directly', link: '/developers/editing-in-typst' },
            { text: 'Releasing', link: '/developers/releasing' },
          ],
        },
      ],
    },
    socialLinks: [{ icon: 'github', link: repo }],
    editLink: {
      pattern: `${repo}/edit/main/docs/:path`,
      text: 'Edit this page on GitHub',
    },
    search: { provider: 'local' },
    outline: [2, 3],
    footer: {
      message: 'Your resumes stay on your computer, in plain files.',
      copyright: 'Icons by <a href="https://lucide.dev">Lucide</a>. Rendering by <a href="https://typst.app">Typst</a>.',
    },
  },
})

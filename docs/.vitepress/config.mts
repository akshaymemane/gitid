import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'GitID',
  description: 'Fast Git identity switching for developers',
  cleanUrls: true,
  lastUpdated: true,
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Commands', link: '/commands/' },
      { text: 'npm', link: 'https://www.npmjs.com/package/@akshaymemane/git-id' }
    ],
    sidebar: [
      {
        text: 'Guide',
        items: [
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Project Philosophy', link: '/guide/philosophy' },
          { text: 'Safety Model', link: '/guide/safety' }
        ]
      },
      {
        text: 'Commands',
        items: [
          { text: 'Overview', link: '/commands/' },
          { text: 'setup', link: '/commands/setup' },
          { text: 'add', link: '/commands/add' },
          { text: 'list and current', link: '/commands/list-current' },
          { text: 'switch', link: '/commands/switch' },
          { text: 'attach and auto', link: '/commands/auto' },
          { text: 'doctor', link: '/commands/doctor' },
          { text: 'backup and restore', link: '/commands/backup-restore' },
          { text: 'Global Options', link: '/commands/options' }
        ]
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/akshaymemane/gitid' }
    ],
    search: {
      provider: 'local'
    },
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright (c) 2026 GitID contributors'
    }
  }
})

import type { Theme } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import FeatureGrid from './FeatureGrid.vue'
import Showcase from './Showcase.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('FeatureGrid', FeatureGrid)
    app.component('Showcase', Showcase)
  },
} satisfies Theme

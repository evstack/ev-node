// https://vitepress.dev/guide/custom-theme
import { h } from 'vue'
import Theme from 'vitepress/theme'
import './style.css'
import { theme } from 'vitepress-openapi/client'
import 'vitepress-openapi/dist/style.css'
import CelestiaGasEstimator from '../components/CelestiaGasEstimator.vue'

export default {
  extends: Theme,
  Layout: () => {
    return h(Theme.Layout, null, {
      // https://vitepress.dev/guide/extending-default-theme#layout-slots
    })
  },
  enhanceApp({ app, router, siteData }) {
    theme.enhanceApp({ app, router, siteData })
    app.component('CelestiaGasEstimator', CelestiaGasEstimator)
  }
}

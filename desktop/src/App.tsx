import { ThemeProvider } from './design-system/ThemeProvider'
import { Shell } from './components/layout/Shell'

export function App() {
  return (
    <ThemeProvider>
      <Shell />
    </ThemeProvider>
  )
}

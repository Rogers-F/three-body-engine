import { memo } from 'react'
import { Card } from '@/components/common/Card'

export const SettingsView = memo(function SettingsView() {
  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">Settings</h2>
      <Card header={<span className="text-sm font-medium text-[var(--text-primary)]">Settings</span>}>
        <p className="text-sm text-[var(--text-muted)]">
          Coming soon — configuration options will appear here.
        </p>
      </Card>
    </div>
  )
})

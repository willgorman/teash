package tui

import (
	"fmt"
	"time"
)

func (m Model) renderStatusBar() string {
	cluster := m.service.ActiveCluster()
	clusterStr := styleCluster.Render(cluster)

	cacheAge := ""
	if updated, err := m.service.CacheLastUpdated(); err == nil && !updated.IsZero() {
		age := time.Since(updated).Truncate(time.Second)
		cacheAge = fmt.Sprintf(" | cache: %s ago", formatDuration(age))
	}

	status := ""
	if m.refreshing {
		status = " | Refreshing..."
	} else if m.status != "" {
		status = " | " + m.status
	}

	errStr := ""
	if m.err != nil {
		errStr = " | " + styleError.Render(m.err.Error())
	}

	count := fmt.Sprintf(" | %d servers", len(m.servers))

	modeStr := ""
	switch m.mode {
	case ModeSearch:
		modeStr = " | MODE: search"
	case ModeAddLabel:
		modeStr = " | MODE: add label"
	case ModeColumnFilter, ModeColumnFilterValue:
		modeStr = " | MODE: column filter"
	}

	help := styleHelp.Render(" | j/k:nav enter:ssh r:refresh a:label /:search c:filter q:quit")

	return styleStatusBar.Render(clusterStr + count + cacheAge + status + modeStr + errStr + help)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

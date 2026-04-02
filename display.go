package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	accent    = lipgloss.Color("#1DA1F2") // twitter blue
	dimColor  = lipgloss.Color("#666666")
	likeColor = lipgloss.Color("#F91880")
	rtColor   = lipgloss.Color("#00BA7C")
	viewColor = lipgloss.Color("#8899A6")
	rankColor = lipgloss.Color("#FFD700")

	// Styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			PaddingBottom(1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			Padding(0, 1).
			MarginBottom(1).
			Width(64)

	rankStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(rankColor)

	handleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	nameStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	verifiedStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true)

	timeStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)

	tweetStyle = lipgloss.NewStyle().
			PaddingTop(1).
			PaddingBottom(1)

	likeStyle = lipgloss.NewStyle().
			Foreground(likeColor)

	rtStyle = lipgloss.NewStyle().
			Foreground(rtColor)

	replyStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	viewStyle = lipgloss.NewStyle().
			Foreground(viewColor)

	mediaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0A458"))
)

func displayTweets(tweets []Tweet, count int) {
	if len(tweets) == 0 {
		fmt.Println(lipgloss.NewStyle().Foreground(dimColor).Render("No tweets found."))
		return
	}

	sort.Slice(tweets, func(i, j int) bool {
		return tweets[i].Engagement > tweets[j].Engagement
	})

	if count > len(tweets) {
		count = len(tweets)
	}
	tweets = tweets[:count]

	title := headerStyle.Render(fmt.Sprintf("  X Feed Highlights  %s", time.Now().Format("Jan 2, 2006")))
	subtitle := lipgloss.NewStyle().Foreground(dimColor).Render(
		fmt.Sprintf("  Top %d by engagement", count))
	fmt.Println()
	fmt.Println(title)
	fmt.Println(subtitle)
	fmt.Println()

	for i, t := range tweets {
		printTweet(i+1, t)
	}
}

func printTweet(rank int, t Tweet) {
	// Top line: rank + handle + name + verified + time
	verified := ""
	if t.IsVerified {
		verified = verifiedStyle.Render(" ✓")
	}

	ago := timeStyle.Render(relativeTime(t.Timestamp))
	handle := handleStyle.Render("@" + t.AuthorHandle)
	name := nameStyle.Render(" " + t.AuthorName)
	r := rankStyle.Render(fmt.Sprintf("#%d", rank))

	topLine := fmt.Sprintf("%s  %s%s%s", r, handle, name, verified)
	// Pad to push time to the right
	padLen := 60 - lipgloss.Width(topLine) - lipgloss.Width(ago)
	if padLen < 1 {
		padLen = 1
	}
	topLine += strings.Repeat(" ", padLen) + ago

	// Tweet body
	lines := wordWrap(t.Text, 58)
	body := tweetStyle.Render(strings.Join(lines, "\n"))

	// Stats line
	stats := buildStats(t)

	card := cardStyle.Render(topLine + "\n" + body + "\n" + stats)
	fmt.Println(card)
}

func buildStats(t Tweet) string {
	parts := []string{
		likeStyle.Render(fmt.Sprintf("♥ %s", formatNumber(t.Likes))),
		rtStyle.Render(fmt.Sprintf("↻ %s", formatNumber(t.Retweets))),
		replyStyle.Render(fmt.Sprintf("💬 %s", formatNumber(t.Replies))),
	}
	if t.Views > 0 {
		parts = append(parts, viewStyle.Render(fmt.Sprintf("👁 %s", formatNumber(t.Views))))
	}
	if t.HasMedia {
		icon := "📷"
		switch t.MediaType {
		case "video":
			icon = "🎬"
		case "animated_gif":
			icon = "GIF"
		}
		parts = append(parts, mediaStyle.Render(icon))
	}
	return strings.Join(parts, "   ")
}

func wordWrap(text string, width int) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	paragraphs := strings.Split(text, "\n")
	var lines []string

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		words := strings.Fields(para)
		if len(words) == 0 {
			continue
		}
		current := words[0]
		for _, w := range words[1:] {
			if len(current)+1+len(w) > width {
				lines = append(lines, current)
				current = w
			} else {
				current += " " + w
			}
		}
		lines = append(lines, current)
	}

	if len(lines) > 5 {
		lines = lines[:5]
		lines[4] += "…"
	}
	return lines
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d"
		}
		return fmt.Sprintf("%dd", days)
	}
}

func formatNumber(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

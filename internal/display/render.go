package display

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"x-digest/internal/twitter"
)

var (
	accent    = lipgloss.Color("#1DA1F2")
	dimColor  = lipgloss.Color("#666666")
	likeColor = lipgloss.Color("#F91880")
	rtColor   = lipgloss.Color("#00BA7C")
	viewColor = lipgloss.Color("#8899A6")
	rankColor = lipgloss.Color("#FFD700")

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

	rankStyle     = lipgloss.NewStyle().Bold(true).Foreground(rankColor)
	handleStyle   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	nameStyle     = lipgloss.NewStyle().Foreground(dimColor)
	verifiedStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	timeStyle     = lipgloss.NewStyle().Foreground(dimColor).Italic(true)
	tweetStyle    = lipgloss.NewStyle().PaddingTop(1).PaddingBottom(1)
	likeStyle     = lipgloss.NewStyle().Foreground(likeColor)
	rtStyle       = lipgloss.NewStyle().Foreground(rtColor)
	replyStyle    = lipgloss.NewStyle().Foreground(dimColor)
	viewStyle     = lipgloss.NewStyle().Foreground(viewColor)
	mediaStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0A458"))
)

const cardInnerWidth = 58

// Tweets sorts by engagement and prints the top count tweets.
func Tweets(tweets []twitter.Tweet, count int) {
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

func printTweet(rank int, t twitter.Tweet) {
	verified := ""
	if t.IsVerified {
		verified = verifiedStyle.Render(" ✓")
	}

	ago := timeStyle.Render(RelativeTime(t.Timestamp))
	handle := handleStyle.Render("@" + t.AuthorHandle)
	name := nameStyle.Render(" " + t.AuthorName)
	r := rankStyle.Render(fmt.Sprintf("#%d", rank))

	topLine := fmt.Sprintf("%s  %s%s%s", r, handle, name, verified)
	padLen := 60 - lipgloss.Width(topLine) - lipgloss.Width(ago)
	if padLen < 1 {
		padLen = 1
	}
	topLine += strings.Repeat(" ", padLen) + ago

	lines := WordWrap(t.Text, cardInnerWidth)
	body := tweetStyle.Render(strings.Join(lines, "\n"))

	stats := buildStats(t)

	fmt.Println(cardStyle.Render(topLine + "\n" + body + "\n" + stats))
}

func buildStats(t twitter.Tweet) string {
	parts := make([]string, 0, 5)
	parts = append(parts,
		likeStyle.Render(fmt.Sprintf("♥ %s", FormatNumber(t.Likes))),
		rtStyle.Render(fmt.Sprintf("↻ %s", FormatNumber(t.Retweets))),
		replyStyle.Render(fmt.Sprintf("💬 %s", FormatNumber(t.Replies))),
	)
	if t.Views > 0 {
		parts = append(parts, viewStyle.Render(fmt.Sprintf("👁 %s", FormatNumber(t.Views))))
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

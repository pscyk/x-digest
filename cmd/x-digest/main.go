package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"x-digest/internal/cookies"
	"x-digest/internal/display"
	"x-digest/internal/twitter"
)

func main() {
	count := flag.Int("count", 20, "number of top tweets to show")
	timeline := flag.String("timeline", "following", "timeline type: 'following' or 'foryou'")
	user := flag.String("user", "", "fetch tweets from a specific user (e.g. elonmusk)")
	bookmarks := flag.String("bookmarks", "", "search your bookmarks (keyword, or 'all' for broad match)")
	profile := flag.String("profile", "", "Firefox profile name override")
	queryID := flag.String("query-id", "", "override GraphQL query ID")
	flag.Parse()

	fmt.Println("Reading Firefox cookies...")
	authToken, ct0, err := cookies.ExtractFirefox(*profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	client := twitter.NewClient(authToken, ct0)
	var tweets []twitter.Tweet

	switch {
	case *bookmarks != "":
		q := *bookmarks
		if q == "all" {
			q = ""
		}
		fmt.Println("Searching bookmarks...")
		tweets, err = client.FetchBookmarks(*count, q)
	case *user != "":
		handle := strings.TrimPrefix(*user, "@")
		fmt.Printf("Fetching tweets from @%s...\n", handle)
		tweets, err = client.FetchUserTweets(handle, *count)
	default:
		if *timeline != "following" && *timeline != "foryou" {
			fmt.Fprintf(os.Stderr, "Error: --timeline must be 'following' or 'foryou'\n")
			os.Exit(1)
		}
		fmt.Printf("Fetching %s timeline...\n", *timeline)
		tweets, err = client.FetchTimeline(*timeline, *count, *queryID)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	display.Tweets(tweets, *count)
}
